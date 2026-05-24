package server

const pageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>DESi Paper Review - Assistant</title>
<style>
  :root { --bg:#0f1115; --panel:#171a21; --ink:#e6e8ec; --muted:#9aa3b2;
          --accent:#5b9dff; --warn:#f0b429; --bad:#ff6b6b; --ok:#4cc38a;
          --border:#262b35; }
  * { box-sizing:border-box; }
  body { margin:0; font:15px/1.55 -apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;
         background:var(--bg); color:var(--ink); }
  header { padding:20px 24px; border-bottom:1px solid var(--border); background:var(--panel); }
  header h1 { margin:0; font-size:20px; }
  header .verdict { display:inline-block; margin-top:8px; padding:2px 10px; border-radius:999px;
                    background:#1d2733; color:var(--accent); font-size:12px; letter-spacing:.04em;
                    border:1px solid var(--border); }
  main { max-width:980px; margin:0 auto; padding:24px; }
  .panel { background:var(--panel); border:1px solid var(--border); border-radius:10px;
           padding:18px 20px; margin-bottom:20px; }
  .panel h2 { margin:0 0 12px; font-size:16px; }
  .muted { color:var(--muted); }
  .health { font-size:13px; }
  .health .dot { display:inline-block; width:9px; height:9px; border-radius:50%; margin-right:6px; }
  .dot.ok { background:var(--ok); } .dot.bad { background:var(--bad); }
  textarea { width:100%; min-height:220px; background:#0c0e12; color:var(--ink);
             border:1px solid var(--border); border-radius:8px; padding:12px; font-family:ui-monospace,SFMono-Regular,Menlo,monospace; font-size:13px; }
  .row { display:flex; gap:12px; align-items:center; flex-wrap:wrap; margin-top:12px; }
  button, .btn { background:var(--accent); color:#08111f; border:0; padding:9px 16px;
                 border-radius:8px; font-weight:600; cursor:pointer; text-decoration:none; font-size:14px; }
  .btn.secondary { background:#222834; color:var(--ink); border:1px solid var(--border); }
  input[type=file] { color:var(--muted); font-size:13px; }
  .error { border-left:3px solid var(--bad); padding-left:12px; color:#ffd0d0; }
  .scope { border-left:3px solid var(--accent); }
  .kv { font-size:13px; color:var(--muted); margin:2px 0; }
  .kv code { color:var(--ink); }
  ul.items { margin:0; padding-left:18px; }
  ul.items li { margin:6px 0; }
  .tag { font-size:11px; padding:1px 7px; border-radius:6px; background:#1d2733; border:1px solid var(--border);
         color:var(--accent); margin-right:6px; }
  .risk { color:var(--warn); font-weight:600; }
  .terms code { color:var(--warn); }
  .counts { display:flex; gap:18px; flex-wrap:wrap; font-size:13px; }
  .counts b { color:var(--accent); }
  pre { background:#0c0e12; border:1px solid var(--border); border-radius:8px; padding:14px;
        overflow:auto; font-size:12px; max-height:420px; }
  details summary { cursor:pointer; color:var(--accent); font-size:14px; }
  footer { color:var(--muted); font-size:12px; text-align:center; padding:24px; }
</style>
</head>
<body>
<header>
  <h1>DESi Paper Review <span class="muted">&mdash; reviewer assistant</span></h1>
  <div class="verdict">REVIEW_ASSISTANCE_ONLY</div>
  <div class="health" style="margin-top:10px">
    {{if .Health}}
      <span class="dot ok"></span>DESi governance online &mdash;
      <span class="muted">{{.Health.Library}} {{.Health.Version}}, core_identity=<code>{{.Health.CoreIdentity}}</code></span>
    {{else}}
      <span class="dot bad"></span><span class="muted">DESi governance service unavailable: {{.HealthErr}}</span>
    {{end}}
  </div>
</header>
<main>

  <div class="panel">
    <h2>Submit a paper</h2>
    <p class="muted">Offline &amp; deterministic. This assists a human reviewer &mdash; it never accepts, rejects, or validates a paper.</p>
    <form method="post" action="/review" enctype="multipart/form-data">
      <textarea name="paper" placeholder="Paste paper text (.md / .txt) here...">{{.Input}}</textarea>
      <div class="row">
        <button type="submit">Review</button>
        <a class="btn secondary" href="/?sample=1">Load sample</a>
        <span class="muted">or upload:</span>
        <input type="file" name="file" accept=".md,.txt,.markdown,text/plain">
      </div>
    </form>
  </div>

  {{if .Error}}
  <div class="panel"><div class="error"><b>Error:</b> {{.Error}}</div></div>
  {{end}}

  {{with .Result}}
  {{$a := .Artifact}}
  <div class="panel scope">
    <h2>Scope</h2>
    <p>{{$a.Governance.Disclaimer}}</p>
    <blockquote class="muted">{{$a.Governance.AuditFraming}}</blockquote>
    <div class="kv">Paper: <code>{{$a.PaperTitle}}</code></div>
    <div class="kv">Verdict: <code>{{$a.Verdict}}</code></div>
    <div class="kv">Governance library: <code>{{$a.Governance.Library}}</code> &middot; core_identity: <code>{{$a.Governance.CoreIdentity}}</code></div>
    <div class="kv">Hype / forbidden-term hits (DESi scan):
      {{if $a.Governance.ForbiddenTermHits}}<code>{{range $a.Governance.ForbiddenTermHits}}{{.}} {{end}}</code>{{else}}<code>none</code>{{end}}</div>
    <div class="kv">Mode: offline_mode=<code>{{$a.Governance.Mode.OfflineMode}}</code>,
      allow_live_llm_calls=<code>{{$a.Governance.Mode.AllowLiveLLMCalls}}</code>,
      live_calls_enabled=<code>{{$a.Governance.Mode.LiveCallsEnabled}}</code></div>
    <div class="kv">Replay hash: <code>{{$a.ReplayHash}}</code></div>
  </div>

  <div class="panel">
    <h2>Summary</h2>
    <div class="counts">
      <span><b>{{len $a.Claims}}</b> claims</span>
      <span><b>{{len $a.Overclaims}}</b> overclaims</span>
      <span><b>{{len $a.EvidenceGaps}}</b> evidence gaps</span>
      <span><b>{{len $a.ReproducibilityRisks}}</b> reproducibility risks</span>
      <span><b>{{len $a.ReviewerQuestions}}</b> reviewer questions</span>
    </div>
  </div>

  <div class="panel">
    <h2>Main Claims</h2>
    {{if $.MainClaims}}
    <ul class="items">
      {{range $.MainClaims}}<li><span class="tag">{{.Category}}</span><span class="muted">[{{.ClaimID}} &middot; {{.Section}}]</span> {{.Text}}</li>{{end}}
    </ul>
    {{else}}<p class="muted">No main claims detected.</p>{{end}}
  </div>

  <div class="panel">
    <h2>Overclaim Risks</h2>
    {{if $a.Overclaims}}
    <ul class="items">
      {{range $a.Overclaims}}<li class="terms"><span class="muted">[{{.ClaimID}} &middot; {{.Section}}]</span> terms: <code>{{range .Terms}}{{.}} {{end}}</code><br>{{.Text}}</li>{{end}}
    </ul>
    {{else}}<p class="muted">No overclaim language detected.</p>{{end}}
  </div>

  <div class="panel">
    <h2>Evidence Gaps</h2>
    {{if $a.EvidenceGaps}}
    <ul class="items">
      {{range $a.EvidenceGaps}}<li><span class="muted">[{{.ClaimID}} &middot; {{.Section}}]</span> {{.Note}}</li>{{end}}
    </ul>
    {{else}}<p class="muted">No evidence gaps flagged.</p>{{end}}
  </div>

  <div class="panel">
    <h2>Reproducibility Risks</h2>
    {{if $a.ReproducibilityRisks}}
    <ul class="items">
      {{range $a.ReproducibilityRisks}}<li><span class="risk">{{.RiskType}}</span> &mdash; {{.Detail}}</li>{{end}}
    </ul>
    {{else}}<p class="muted">No reproducibility risks flagged.</p>{{end}}
  </div>

  <div class="panel">
    <h2>Questions for Human Reviewer</h2>
    <ul class="items">
      {{range $a.ReviewerQuestions}}<li>{{.}}</li>{{end}}
    </ul>
  </div>

  <div class="panel">
    <h2>Artifacts</h2>
    <div class="row" style="margin-bottom:14px">
      <form method="post" action="/download">
        <input type="hidden" name="paper" value="{{$.Input}}">
        <input type="hidden" name="format" value="json">
        <button type="submit">Download JSON</button>
      </form>
      <form method="post" action="/download">
        <input type="hidden" name="paper" value="{{$.Input}}">
        <input type="hidden" name="format" value="md">
        <button class="btn secondary" type="submit">Download report (.md)</button>
      </form>
    </div>
    <details><summary>JSON artifact (DESi-canonical, replay-stable)</summary><pre>{{.JSON}}</pre></details>
    <details><summary>Markdown report</summary><pre>{{$.ReportMarkdown}}</pre></details>
  </div>
  {{end}}

</main>
<footer>DESi Paper Review &middot; reviewer assistant, not a peer reviewer &middot; governance via the real desi-governance library</footer>
</body>
</html>
`
