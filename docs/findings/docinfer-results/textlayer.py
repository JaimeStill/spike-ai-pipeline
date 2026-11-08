"""textlayer.py <docproc-dir> <docinfer-dir> <out.json> [per-input N]: for document pages that
have both a page image and a text-layer output (paired by part SHA-256 and page meta), the word
recall of the vision text against the text layer, and the words vision has that the layer
lacks. Words are runs of letters or digits, lowercased, compared as multisets. Pages with an
empty text layer are counted apart. With N, a seeded sample of N pages per named input is also
written in full for reading.

out.json holds the summary and the sample; the per-page rows go beside it, to
<out>-rows.json. The summary also derives, each with its rule written into it, the pages
whose layer is thin, the text-heavy pages vision described instead of transcribing, and the
page images with no text output. The first two leave out text_heavy.pdf, a synthetic PDF of
random letter strings whose recall measures nothing a real document has."""
import collections, glob, hashlib, json, os, random, re, sys

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
no_layer_words = collections.Counter()  # their vision texts' words, by input
for (sha, pg), v in pages.items():
    if "image" not in v:
        continue
    c = hashlib.sha256(open(P + "/" + v["image"], "rb").read()).hexdigest()
    r = held.get(c)
    if "text" not in v:
        no_layer[v["input"]] += 1
        if r and r.get("text"):
            no_layer_words[v["input"]] += sum(words(open(D + "/" + r["text"], encoding="utf-8").read()).values())
        continue
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
                     incomplete=r.get("incomplete", ""), extra=[w for w, _ in extra.most_common(25)],
                     opening=vis.lstrip(" \t\n*#")[:200]))
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
real = [r for r in rows if r["input"] != "text_heavy.pdf"]
# A thin layer: the vision text has at least twice the layer's words and at least 60 more.
thin = [r for r in real if r["vision_words"] >= 2 * r["layer_words"] and r["vision_words"] - r["layer_words"] >= 60]
summary["thin_layer"] = dict(
    rule="pages other than text_heavy.pdf's whose vision text has at least twice the layer's words and at least 60 more",
    pages=len(thin), layer_words=sum(r["layer_words"] for r in thin), vision_words=sum(r["vision_words"] for r in thin),
    by_input=dict(collections.Counter(r["input"] for r in thin).most_common()))
# Described: a page whose layer holds a page of text, whose vision text recalls under half of
# it, and whose vision text opens by describing the image rather than with the page's words.
DESCRIBES = re.compile(r"(this|the) (image|picture)\b|this is an image\b|an? [^.\n]{0,80}?\b(shows|contains|depicts)\b", re.I)
heavy = [r for r in real if r["layer_words"] >= 100]
described = [r for r in heavy if r["recall"] < 0.5 and DESCRIBES.match(r["opening"])]
summary["described"] = dict(
    rule="pages other than text_heavy.pdf's with at least 100 layer words, recall under 0.5, and a vision text "
         "that opens by describing the image: " + DESCRIBES.pattern + " (case-insensitive, matched at the start "
         "after leading whitespace, * and #)",
    text_heavy_pages=len(heavy), pages=len(described),
    by_input=dict(collections.Counter(r["input"] for r in described).most_common()),
    list=[[r["input"], r["page"], round(r["recall"], 3)] for r in described])
summary["no_layer"] = dict(
    rule="page images with no text output for their page; the words of their vision texts",
    pages=sum(no_layer.values()), vision_words=sum(no_layer_words.values()),
    pages_without_text_heavy=sum(n for k, n in no_layer.items() if k != "text_heavy.pdf"),
    vision_words_without_text_heavy=sum(n for k, n in no_layer_words.items() if k != "text_heavy.pdf"))
for k, rs in sorted(by.items(), key=lambda kv: -len(kv[1])):
    summary[k] = stats(rs) | {"empty_layer": sum(1 for r in rs if r["layer_words"] == 0)}
random.seed(7)
sample = []
for name in ("cfr-2024-title32-vol1.pdf", "dodm-5200-01.pdf", "text_heavy.pdf"):
    rs = [r for r in by.get(name, []) if r["layer_words"]]
    sample += random.sample(rs, min(per, len(rs)))
json.dump(dict(summary=summary, sample=sample), open(out, "w"), ensure_ascii=False, indent=1)
json.dump(rows, open(os.path.splitext(out)[0] + "-rows.json", "w"), ensure_ascii=False, indent=1)
print(json.dumps(summary, indent=1))
for r in sample:
    print(f'{r["input"][:28]:28} p{r["page"]:4} layer {r["layer_words"]:4} vision {r["vision_words"]:4} recall {r["recall"]:.2f} prec {r["precision"]:.2f} {r["incomplete"][:4]} extra {r["extra"][:8]}')
