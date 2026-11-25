# 60db Go SDK

Official Go client for the [60db](https://60db.ai) API. Standard library only, Go 1.21+.

```bash
go get github.com/60db-ai/60db-go
```

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"errors"
	"os"

	sixtydb "github.com/60db-ai/60db-go"
)

func main() {
	ctx := context.Background()
	c, _ := sixtydb.NewClient(sixtydb.Config{APIKey: os.Getenv("SIXTYDB_API_KEY")})

	// Text to speech -> complete WAV
	wav, err := c.TextToSpeech(ctx, "Hello world", "", map[string]any{"sample_rate": 24000})
	if err != nil { panic(err) }
	os.WriteFile("out.wav", wav, 0o644)

	// Streaming TTS -> 16-bit LE mono PCM chunks
	var pcm []byte
	_ = c.TextToSpeechStream(ctx, "Streaming", "", map[string]any{"sample_rate": 24000},
		func(p []byte) error { pcm = append(pcm, p...); return nil })
	os.WriteFile("stream.wav", sixtydb.PCMToWAV(pcm, 24000), 0o644)

	// Chat (streaming)
	_ = c.Chat.Stream(ctx, sixtydb.ChatRequest{Messages: []sixtydb.ChatMessage{{Role: "user", Content: "Hi"}}},
		func(chunk map[string]any) error { fmt.Print(sixtydb.ChunkContent(chunk)); return nil })

	// Errors
	if _, err := c.GetProfile(ctx); err != nil {
		var e *sixtydb.Error
		_ = errors.As(err, &e) // e.Status, e.Code, e.Field, e.Message, e.Body
	}
}
```

JSON results are `map[string]any`. Extra API fields go in `params map[string]any`.
Every network method takes `context.Context` first.

## Methods

| Area | Methods |
|---|---|
| TTS | `TextToSpeech`, `TextToSpeechStream`, `PCMToWAV` |
| STT | `SpeechToText`, `SpeechToTextFile` |
| Voices | `GetVoices`, `GetVoice`, `GetMyVoices`, `CreateVoice`, `UpdateVoice`, `DeleteVoice` |
| Languages / models | `GetLanguages` (STT), `GetTTSLanguages`, `GetTTSModels`, `GetSTTModels` |
| Chat | `Chat.Completion`, `Chat.Stream`, `ChunkContent` |
| Memory | `Memory.Ingest`, `Memory.Search`, `Memory.Context`, `Memory.ExtractDocument(File)`, `Memory.Usage` |
| Webhooks | `ComputeWebhookSignature`, `VerifyWebhookSignature` |
| Account | `GetProfile`, `UpdateProfile`, `GetWorkspaces`, `CreateWorkspace`, `GetPlans`, `GetCurrentPlan`, `Subscribe`, `GetUsage`, `GetAPIKeys`, `CreateAPIKey`, `DeleteAPIKey`, `GetWebhooks`, `CreateWebhook`, `DeleteWebhook`, `SignUp`, `SignIn` |

| Judge | `Judge.Evaluate`, `Extract`, `Models`, `Usage`, `Health`; `Judge.Rubrics.{List,Create,Get,Update,Delete}`; `Judge.Runs.{List,Get,Delete}` (results unwrap `data`) |
| Music | `Music.Create` (JSON or lyrics file), `List`, `Get`, `Update`, `Delete`, `Download`, `WaitForCompletion`, `ListVoices`, `CreateVoice`, `Health` |
| Agents | `Agents.{List,Get,Create,Update,Delete,RotateWebhookSecret,ToolEvents,Call,StreamCall,StreamCallWithParams,WebCall,BackgroundSounds}`; `Agents.Settings.{Get,Update,RotateWebhookSecret}`; `Agents.Calls.{List,Get,PlaceInline,Usage,ToolEvents,Transcript,Recording,DeleteRecording,Result,EndStream,HangUpWeb,WaitForCompletion}`; `Agents.PhoneLines.{List,AvailableNumbers,Get,Create,Update,Delete,Setup,Check}`; `Agents.Builder.{CreateDraft,GetDraft,WaitForDraft}` |
| Campaigns | `Agents.Campaigns.{List,Summary,PreviewRecipients,Create,Get,Analytics,Pause,Resume,Cancel}`; `Agents.Campaigns.Recipients.{List,Add,Preview,Template,Export,Calls,CancelCallback,CallbackNow}`; `Agents.DoNotCall.{List,Add,Remove,RemoveByPhone}` |
| Integrations | `Agents.Integrations.{Catalog,List,Create,Update,Delete,Test,RotateSecret,Tools,AssignAgents,RunAction,Request,GoogleSheetTabs,GoogleSheetColumns}` |
| Simulations | `Agents.Simulations.{List,Start,Estimate,Get,WaitForRun,Pause,Resume,Cancel,CreateDraft,UpdateDraft,Rerun,RerunCall,SuggestFix,ApplyFix,UndoFix,Evals,Metrics,CompareMetrics,RunGolden,GetGoldenRun}` |
| Call analysis | `CallAnalysis.Analyzers.{List,Create,Get,Update,Archive}`; `CallAnalysis.{DialerNumbers,AnalyzeAgentCalls,AnalyzeDialerCalls,AnalyzeAll,Recordings,Problems,Analytics,Overview,GetCall}`; `CallAnalysis.Sources.{Get,Set}`; `CallAnalysis.Batches.{Create,Get,Items,CreateUploadSlot,ConfirmUpload,FetchRecording,FetchRecordings,Analyze}`; `CallAnalysis.AgentFix.{Suggest,Apply,Undo}` |
| Dialer | `Dialer.Status`; `Dialer.Numbers.*`, `Dialer.Calls.*`, `Dialer.Recordings.*`, `Dialer.Billing.*`, `Dialer.Webhooks.{List,Create,Update,Delete,RotateSecret,Test,Deliveries}` |

Judge, Agents and Call analysis methods return the API's `data` as `any` (a `map[string]any` or `[]any`); failures are
`*sixtydb.Error` (e.g. `Status 403, Code "POLICY_DENY"` when the key lacks the `agents` scope). Music and Dialer return the
full body. Paged lists (`Campaigns.List`, `Campaigns.Recipients.List`, `DoNotCall.List`, `Simulations.List`,
`CallAnalysis.Batches.Items`) return a `*sixtydb.Page` with `Data` (the rows) and `Pagination` (`total`, `limit`, `offset`).
CSV downloads (`Recipients.Template`, `Recipients.Export`) return the CSV as a `string`.

## Agents

Placing calls (`Agents.Call`, `Calls.PlaceInline`), starting or resuming campaigns and adding recipients ring real phones
and spend credits. Writes that dial take an optional idempotency key (`""` for none) sent as `Idempotency-Key`, so a retry
does not dial twice.

```go
// Call with fill-ins for {{name}} in the script, then wait for the end.
call, _ := c.Agents.Call(ctx, agentID, sixtydb.PlaceCallParams{To: "+15550001", PhoneLineID: lineID,
	Variables: map[string]string{"name": "Asha"}})
done, _ := c.Agents.Calls.WaitForCompletion(ctx, call.(map[string]any)["id"].(string), nil)
usage, _ := c.Agents.Calls.Usage(ctx, done["id"].(string)) // measured, total, services, turns, phone, price_usd

// A call that carries its own agent (settings.max_duration_secs is required).
_, _ = c.Agents.Calls.PlaceInline(ctx, map[string]any{"to": "+15550001", "phone_line_id": lineID,
	"agent":    map[string]any{"script": "You confirm appointments.", "voice_id": voiceID},
	"settings": map[string]any{"max_duration_secs": 300}}, "confirm-42")

// Stream and browser calls
s, _ := c.Agents.StreamCallWithParams(ctx, agentID, sixtydb.StreamCallParams{Variables: map[string]string{"name": "Asha"}})
w, _ := c.Agents.WebCall(ctx, agentID, offerSDP) // {call, answer}
_, _ = c.Agents.Calls.EndStream(ctx, callID)    // or Calls.HangUpWeb for a web call

// Workspace webhooks for calls without a stored agent (nil clears one); BYO phone line checks
_, _ = c.Agents.Settings.Update(ctx, map[string]any{"event_webhook_url": "https://example.com/60db", "tool_webhook_url": nil})
guide, _ := c.Agents.PhoneLines.Setup(ctx, lineID)
_, _ = c.Agents.PhoneLines.Check(ctx, lineID, map[string]any{"called_number": "+15550001", "source_ip": "203.0.113.7"})
```

`Calls.List` also filters by `Sentiment`. `PhoneLines.Create`/`Update` pass BYO trunk fields through as given
(`numbers`, `allowed_addresses`, `allowed_numbers`, `normalize_numbers`, `default_country_code`, `bridge_transfers`;
`username`/`password` are optional for an IP-trust carrier).

### AI agent builder

```go
d, _ := c.Agents.Builder.CreateDraft(ctx, sixtydb.AgentDraftParams{
	Description: "Books dental appointments", Direction: "inbound", WebsiteURL: "https://clinic.example",
	FilePaths: []string{"faq.pdf"},                                         // or Files: []sixtydb.UploadFile{{Filename, Data}}
})
draft, _ := c.Agents.Builder.WaitForDraft(ctx, d.(map[string]any)["id"].(string), nil) // ready or failed (not an error)
if draft["status"] == "ready" {
	agent := draft["draft"].(map[string]any)["agent"].(map[string]any) // a ready Agents.Create payload
	_, _ = c.Agents.Create(ctx, agent)
}
```

Up to 5 documents (pdf, docx, doc, txt, md, rtf, html, csv, xlsx, pptx; 20 MB each) go under the form field `files`.

### Campaigns and do-not-call

```go
check, _ := c.Agents.Campaigns.PreviewRecipients(ctx, map[string]any{"csv_text": csv, "agent_id": agentID})
camp, _ := c.Agents.Campaigns.Create(ctx, map[string]any{"name": "Renewals", "agent_id": agentID,
	"phone_line_id": lineID, "timezone": "Asia/Kolkata", "csv_text": csv,
	"calling_windows": []map[string]any{{"days": []int{1, 2, 3, 4, 5}, "start": "10:00", "end": "18:00"}}}, "renewals-2026-11")
id := camp.(map[string]any)["id"].(string)
page, _ := c.Agents.Campaigns.Recipients.List(ctx, id, map[string]any{"status": "callback", "limit": 50})
fmt.Println(len(page.Data), page.Pagination["total"])
csvOut, _ := c.Agents.Campaigns.Recipients.Export(ctx, id, map[string]any{"status": "completed"})
_, _ = c.Agents.Campaigns.Pause(ctx, id)

_, _ = c.Agents.DoNotCall.Add(ctx, map[string]any{"phone": "+919876543210", "reason": "asked"})
_, _ = c.Agents.DoNotCall.RemoveByPhone(ctx, "+919876543210")
```

### Integrations

Key, webhook and MCP providers connect with `Integrations.Create` (credentials are tested live). OAuth providers are
connected in the 60db dashboard: the OAuth callback is bound to the browser that started it, so the SDK has no OAuth call.

```go
cat, _ := c.Agents.Integrations.Catalog(ctx)
conn, _ := c.Agents.Integrations.Create(ctx, map[string]any{"provider": "hubspot", "name": "HubSpot (sales)",
	"fields": map[string]any{"token": hubspotToken}})
cid := conn.(map[string]any)["id"].(string)
tools, _ := c.Agents.Integrations.Tools(ctx, cid)
_, _ = c.Agents.Integrations.AssignAgents(ctx, cid, []map[string]any{{"agent_id": agentID, "tools": []string{"search_contact"}}})
res, _ := c.Agents.Integrations.RunAction(ctx, cid, "search_contact", map[string]any{"email": "jane@acme.com"}) // {ok, result}
```

### Simulations, evals and metrics

Starting, resuming and re-running simulations, golden runs and fix suggestions spend credits (`Estimate` is free).

```go
est, _ := c.Agents.Simulations.Estimate(ctx, agentID, map[string]any{"repeats": 1, "max_credits": 200, "estimated_credits_per_call": 4})
run, _ := c.Agents.Simulations.Start(ctx, agentID, map[string]any{"mode": "generate", "count": 10,
	"max_credits": 200, "estimated_credits_per_call": 4})
done, _ := c.Agents.Simulations.WaitForRun(ctx, run.(map[string]any)["run_id"].(string), nil) // finished, cancelled or stopped_budget
fix, _ := c.Agents.Simulations.SuggestFix(ctx, runID)                                          // saves nothing
m, _ := c.Agents.Simulations.Metrics(ctx, agentID, map[string]any{"source": "production", "agent_version": 3})
```

Waiters take `*sixtydb.AgentWaitOptions{Interval, Timeout}`; defaults are 2 s / 20 min (`WaitForCompletion`),
2 s / 5 min (`WaitForDraft`) and 4 s / 30 min (`WaitForRun`). They retry 429, 5xx and network errors and return an
`*Error` with `Code "TIMEOUT"` when the time runs out.

## Call analysis

`c.CallAnalysis` needs a user API key with the `call_analysis` permission (else 403 `DA_API_KEY_SCOPE_MISSING`); ids are
UUIDs and the limit is 300 requests a minute per key. An analyzer is an analysis agent.

```go
az, _ := c.CallAnalysis.Analyzers.Create(ctx, map[string]any{"name": "Support QA", "instructions": "Flag refund promises."})
aid := az.(map[string]any)["id"].(string)
_, _ = c.CallAnalysis.Sources.Set(ctx, aid, map[string]any{"agent_ids": []string{agentID}, "min_seconds": 30})
_, _ = c.CallAnalysis.AnalyzeAgentCalls(ctx, aid, []string{callID})

// Recordings in via CSV, then analyze them
b, _ := c.CallAnalysis.Batches.Create(ctx, aid, map[string]any{"source_kind": "csv_urls", "csv_text": csv})
bid := b.(map[string]any)["id"].(string)
items, _ := c.CallAnalysis.Batches.Items(ctx, bid, map[string]any{"limit": 100}) // *Page
_, _ = c.CallAnalysis.Batches.Analyze(ctx, aid, bid, len(items.Data))

probs, _ := c.CallAnalysis.Problems(ctx, aid, nil)
one, _ := c.CallAnalysis.GetCall(ctx, callID)
```

## Tests

```bash
go test ./...                              # offline
SIXTYDB_API_KEY=... go test -run Live -v   # live (TestLive, TestLivePhase2; uses real credits)
```
