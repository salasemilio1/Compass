"""The Q2 arms (protocol: experiments/2026-10-22-retrieval-q2-keywords.md). Each is f(query) -> [paths]."""
import json, pathlib, sys
HERE = pathlib.Path(__file__).parent
sys.path.insert(0, str(HERE.parent))
import methods as M, emb as E, fusion as F, hygiene as H

plain = json.loads((HERE / "c127-q2.json").read_text())
withkw = json.loads((HERE / "c127kw-q2.json").read_text())
bm25f = M.BM25F(plain["records"], 3.0)
hyg = H.Hygiene(plain["records"])
hygkw = H.Hygiene(withkw["records"], keywords=True)
_index = E.Index(E.Static("potion-retrieval-32M", 256), plain["records"], "full")

ARMS = {
    "rg": lambda q: plain["rankings"][q]["keyword"],
    "rg+kw": lambda q: withkw["rankings"][q]["keyword"],
    "fts5": lambda q: plain["rankings"][q]["bm25"],
    "fts5+kw": lambda q: withkw["rankings"][q]["bm25"],
    "bm25f": bm25f.rank,
    "hygiene": hyg.rank,
    "hygiene+kw": hygkw.rank,
    "blend": lambda q: F.convex(bm25f, _index, q, 0.5),
}
