"""textlayer.py <docproc-dir> <docinfer-dir> <out.json> [per-input N]: for document pages that
have both a page image and a text-layer output (paired by part SHA-256 and page meta), the word
recall of the vision text against the text layer, and the words vision has that the layer
lacks. Words are runs of letters or digits, lowercased, compared as multisets. Pages with an
empty text layer are counted apart. With N, a seeded sample of N pages per named input is also
written in full for reading."""
import collections, glob, hashlib, json, random, re, sys

P, D, out = sys.argv[1], sys.argv[2], sys.argv[3]
per = int(sys.argv[4]) if len(sys.argv) > 4 else 10
rec = sorted(glob.glob(P + "/runs/*/record.jsonl"))[0]
pages = collections.defaultdict(dict)  # (part sha, page) -> {image, text, input}
for l in open(rec):
    e = json.loads(l)
    if e["event"] != "output" or e.get("stage", {}).get("name") != "document":
        continue
    o = e["output"]
    pg = (o.get("meta") or {}).get("page")
    if pg is None or o["kind"] not in ("image", "text"):
        continue
    k = (e["part"]["sha256"], pg)
    pages[k][o["kind"]] = o["ref"]
    pages[k]["input"] = e["part"]["name"]
held = {}
for f in glob.glob(D + "/results/vision/*/*.json"):
    r = json.load(open(f))
    held[r["content"]] = r
words = lambda s: collections.Counter(w.lower() for w in re.findall(r"[^\W_]+", s))
rows, sums = [], {}
no_layer = collections.Counter()  # pages with an image and no text output, by input
for (sha, pg), v in pages.items():
    if "image" in v and "text" not in v:
        no_layer[v["input"]] += 1
    if "image" not in v or "text" not in v:
        continue
    c = hashlib.sha256(open(P + "/" + v["image"], "rb").read()).hexdigest()
    r = held.get(c)
    if not r or not r.get("text"):
        continue
    layer = open(P + "/" + v["text"], encoding="utf-8", errors="replace").read()
    vis = open(D + "/" + r["text"], encoding="utf-8").read()
    a, b = words(layer), words(vis)
    na, nb, inter = sum(a.values()), sum(b.values()), sum((a & b).values())
    extra = b - a
    rows.append(dict(input=v["input"], page=int(pg), image=v["image"], text=v["text"], vision=r["text"],
                     layer_words=na, vision_words=nb, shared=inter,
                     recall=(inter / na if na else None), precision=(inter / nb if nb else None),
                     incomplete=r.get("incomplete", ""), extra=[w for w, _ in extra.most_common(25)]))
rows.sort(key=lambda r: (r["input"], r["page"]))
def stats(rs):
    rs = [r for r in rs if r["recall"] is not None]
    if not rs:
        return {}
    rec_ = sorted(r["recall"] for r in rs)
    tot = sum(r["layer_words"] for r in rs)
    return dict(pages=len(rs), recall_median=round(rec_[len(rec_) // 2], 3), recall_p10=round(rec_[len(rec_) // 10], 3),
                recall_pooled=round(sum(r["shared"] for r in rs) / tot, 3),
                precision_pooled=round(sum(r["shared"] for r in rs) / max(1, sum(r["vision_words"] for r in rs)), 3),
                below_0_8=sum(1 for x in rec_ if x < 0.8))
by = collections.defaultdict(list)
for r in rows:
    by[r["input"]].append(r)
summary = {"all": stats(rows), "empty_layer_pages": sum(1 for r in rows if r["layer_words"] == 0),
           "empty_layer_vision_words": sum(r["vision_words"] for r in rows if r["layer_words"] == 0),
           "no_layer_pages": dict(no_layer.most_common())}
for k, rs in sorted(by.items(), key=lambda kv: -len(kv[1])):
    summary[k] = stats(rs) | {"empty_layer": sum(1 for r in rs if r["layer_words"] == 0)}
random.seed(7)
sample = []
for name in ("cfr-2024-title32-vol1.pdf", "dodm-5200-01.pdf", "text_heavy.pdf"):
    rs = [r for r in by.get(name, []) if r["layer_words"]]
    sample += random.sample(rs, min(per, len(rs)))
json.dump(dict(summary=summary, sample=sample, rows=rows), open(out, "w"), ensure_ascii=False, indent=1)
print(json.dumps(summary, indent=1))
for r in sample:
    print(f'{r["input"][:28]:28} p{r["page"]:4} layer {r["layer_words"]:4} vision {r["vision_words"]:4} recall {r["recall"]:.2f} prec {r["precision"]:.2f} {r["incomplete"][:4]} extra {r["extra"][:8]}')
