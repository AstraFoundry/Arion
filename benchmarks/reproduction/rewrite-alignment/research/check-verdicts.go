package main

import (
 "bufio"
 "encoding/json"
 "fmt"
 "os"

 "arion.local/core/domain"
)

func main() {
 data, err := os.ReadFile(os.Args[1]); if err != nil { panic(err) }
 var corpus struct { Samples []struct { ID string; ApprovedMemory []string } }
 if err := json.Unmarshal(data, &corpus); err != nil { panic(err) }
 terms := map[string][]string{}
 for _, s := range corpus.Samples { terms[s.ID] = s.ApprovedMemory }
 file, err := os.Open(os.Args[2]); if err != nil { panic(err) }; defer file.Close()
 scanner := bufio.NewScanner(file)
 checked := 0
 for scanner.Scan() {
  var r struct { CaseID, Source, Proposal, Verdict string; Truncated, Bypassed bool }
  if err := json.Unmarshal(scanner.Bytes(), &r); err != nil { panic(err) }
  if r.Bypassed { continue }
  verdict := domain.ValidateRewrite(domain.RewriteCheck{Source:r.Source,Candidate:r.Proposal,Truncated:r.Truncated,Terms:terms[r.CaseID]})
  if string(verdict) != r.Verdict { panic(fmt.Sprintf("%s changed verdict: %s -> %s",r.CaseID,r.Verdict,verdict)) }
  checked++
 }
 if err := scanner.Err(); err != nil { panic(err) }
 fmt.Printf("Current validator matches all %d retained final verdicts\n",checked)
}
