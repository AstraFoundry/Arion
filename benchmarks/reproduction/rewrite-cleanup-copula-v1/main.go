package main

import (
 "encoding/json"
 "fmt"
 "os"

 "arion.local/core/domain"
)

type input struct { ID string `json:"id"`; Source string `json:"source"` }
type observation struct { ID string `json:"id"`; Source string `json:"source"`; Cleanup domain.SpeechCleanup `json:"cleanup"` }
func main() {
 data, err := os.ReadFile(os.Args[1]); if err != nil { panic(err) }
 var inputs []input; if err := json.Unmarshal(data, &inputs); err != nil { panic(err) }
 result := make([]observation, 0, len(inputs))
 for _, item := range inputs {
  cleanup := domain.CleanSpeech(item.Source)
  text, err := domain.ApplySpeechSpans(item.Source, cleanup.Spans); if err != nil { panic(err) }
  if text != cleanup.Text { panic("cleanup span replay mismatch") }
  result = append(result, observation{ID:item.ID, Source:item.Source, Cleanup:cleanup})
 }
 encoded, err := json.MarshalIndent(result,"","  "); if err != nil { panic(err) }; fmt.Println(string(encoded))
}
