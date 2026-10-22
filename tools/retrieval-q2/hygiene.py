"""The Q2 "hygiene" ranker, frozen before Q2 is scored (protocol: experiments/2026-10-22-retrieval-q2-keywords.md).

BM25F (methods.BM25F's scoring, claim + filename x3) with the three fixes the old-set failure
analysis named, parameters fixed from that analysis and never tuned on Q2:
- a real Porter (1980) stemmer instead of the suffix stripper ("hedged" and "hedge" meet);
- hyphenated compounds kept whole as an extra token ("what-if" no longer splits into stopwords);
- a x1.15 score prior for decisions/ records (hub docs stopped outranking them).
With keywords=True, a record's `keywords:` frontmatter line also counts like the claim.
"""
import math
import re
from collections import Counter, defaultdict

import numpy as np

import methods

DECISION_PRIOR = 1.15
CLAIM_WEIGHT = 3.0


def _cons(word, i):
    ch = word[i]
    if ch in "aeiou":
        return False
    if ch == "y":
        return i == 0 or not _cons(word, i - 1)
    return True


def _m(stem):
    """Porter's measure: the number of VC sequences in stem."""
    n, i, length = 0, 0, len(stem)
    while i < length and _cons(stem, i):
        i += 1
    while i < length:
        while i < length and not _cons(stem, i):
            i += 1
        if i >= length:
            break
        n += 1
        while i < length and _cons(stem, i):
            i += 1
    return n


def _has_vowel(stem):
    return any(not _cons(stem, i) for i in range(len(stem)))


def _double_cons(word):
    return len(word) >= 2 and word[-1] == word[-2] and _cons(word, len(word) - 1)


def _cvc(word):
    if len(word) < 3:
        return False
    return _cons(word, len(word) - 3) and not _cons(word, len(word) - 2) and _cons(word, len(word) - 1) and word[-1] not in "wxy"


def _replace(word, suffix, replacement, condition):
    if word.endswith(suffix):
        stem = word[: len(word) - len(suffix)]
        if condition(stem):
            return stem + replacement, True
        return word, True
    return word, False


def porter(word):
    if len(word) <= 2:
        return word
    # Step 1a
    for suffix, replacement in (("sses", "ss"), ("ies", "i"), ("ss", "ss"), ("s", "")):
        if word.endswith(suffix):
            word = word[: len(word) - len(suffix)] + replacement
            break
    # Step 1b
    extra = False
    if word.endswith("eed"):
        if _m(word[:-3]) > 0:
            word = word[:-1]
    else:
        for suffix in ("ed", "ing"):
            if word.endswith(suffix) and _has_vowel(word[: -len(suffix)]):
                word, extra = word[: -len(suffix)], True
                break
    if extra:
        if word.endswith(("at", "bl", "iz")):
            word += "e"
        elif _double_cons(word) and word[-1] not in "lsz":
            word = word[:-1]
        elif _m(word) == 1 and _cvc(word):
            word += "e"
    # Step 1c
    if word.endswith("y") and _has_vowel(word[:-1]):
        word = word[:-1] + "i"
    # Step 2
    for suffix, replacement in (("ational", "ate"), ("tional", "tion"), ("enci", "ence"), ("anci", "ance"),
                                ("izer", "ize"), ("abli", "able"), ("alli", "al"), ("entli", "ent"), ("eli", "e"),
                                ("ousli", "ous"), ("ization", "ize"), ("ation", "ate"), ("ator", "ate"),
                                ("alism", "al"), ("iveness", "ive"), ("fulness", "ful"), ("ousness", "ous"),
                                ("aliti", "al"), ("iviti", "ive"), ("biliti", "ble")):
        word, matched = _replace(word, suffix, replacement, lambda s: _m(s) > 0)
        if matched:
            break
    # Step 3
    for suffix, replacement in (("icate", "ic"), ("ative", ""), ("alize", "al"), ("iciti", "ic"),
                                ("ical", "ic"), ("ful", ""), ("ness", "")):
        word, matched = _replace(word, suffix, replacement, lambda s: _m(s) > 0)
        if matched:
            break
    # Step 4
    for suffix in ("al", "ance", "ence", "er", "ic", "able", "ible", "ant", "ement", "ment", "ent",
                   "ion", "ou", "ism", "ate", "iti", "ous", "ive", "ize"):
        if word.endswith(suffix):
            stem = word[: len(word) - len(suffix)]
            if _m(stem) > 1 and (suffix != "ion" or stem.endswith(("s", "t"))):
                word = stem
            break
    # Step 5
    if word.endswith("e"):
        stem = word[:-1]
        if _m(stem) > 1 or (_m(stem) == 1 and not _cvc(stem)):
            word = stem
    if _m(word) > 1 and _double_cons(word) and word.endswith("l"):
        word = word[:-1]
    return word


COMPOUND = re.compile(r"[a-z0-9]+(?:-[a-z0-9]+)+")


def tokens(text):
    text = text.lower()
    out = [compound for compound in COMPOUND.findall(text)]
    for raw in re.findall(r"[a-z0-9]+", text):
        if raw in methods.STOP or len(raw) < 2:
            continue
        out.append(porter(raw))
    return out


KEYWORDS_LINE = re.compile(r"(?m)^keywords:\s*(.+)$")


class Hygiene:
    def __init__(self, records, keywords=False, k1=1.2, b=0.75):
        self.paths = [r["path"] for r in records]
        self.tf = []
        for r in records:
            title = r["path"].rsplit("/", 1)[-1].removesuffix(".md").replace("-", " ")
            claim = r["claim"] + " " + title
            if keywords:
                match = KEYWORDS_LINE.search(r["body"])
                if match:
                    claim += " " + match.group(1)
            counts = Counter(tokens(r["body"]))
            for t in tokens(claim):
                counts[t] += CLAIM_WEIGHT
            self.tf.append(counts)
        self.lengths = np.array([sum(c.values()) for c in self.tf])
        self.avg = self.lengths.mean()
        df = Counter(t for c in self.tf for t in c)
        n = len(records)
        self.idf = {t: math.log(1 + (n - d + 0.5) / (d + 0.5)) for t, d in df.items()}
        self.k1, self.b = k1, b
        self.postings = defaultdict(list)
        for i, c in enumerate(self.tf):
            for t, v in c.items():
                self.postings[t].append((i, v))
        self.prior = np.array([DECISION_PRIOR if p.startswith("decisions/") else 1.0 for p in self.paths])

    def rank(self, query):
        s = np.zeros(len(self.paths))
        for t, w in Counter(tokens(query)).items():
            idf = self.idf.get(t)
            if idf is None:
                continue
            for i, v in self.postings[t]:
                norm = self.k1 * (1 - self.b + self.b * self.lengths[i] / self.avg)
                s[i] += w * idf * v * (self.k1 + 1) / (v + norm)
        s = s * self.prior
        return [self.paths[i] for i in np.argsort(-s, kind="stable") if s[i] > 0]
