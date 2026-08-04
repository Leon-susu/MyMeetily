package modelmanager

type CatalogItem struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	FileName    string `json:"fileName,omitempty"`
	URL         string `json:"url,omitempty"`
	Size        int64  `json:"size"`
	Description string `json:"description"`
}

var whisperCatalog = []CatalogItem{
	{ID: "tiny", Kind: "whisper", Name: "Whisper Tiny", FileName: "ggml-tiny.bin", URL: "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-tiny.bin", Size: 77_700_000, Description: "最快，適合較舊或低功耗 CPU。"},
	{ID: "base", Kind: "whisper", Name: "Whisper Base", FileName: "ggml-base.bin", URL: "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.bin", Size: 148_000_000, Description: "速度優先，短會議與清楚語音適用。"},
	{ID: "small", Kind: "whisper", Name: "Whisper Small", FileName: "ggml-small.bin", URL: "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin", Size: 488_000_000, Description: "CPU 筆電的速度與準確度平衡選擇。"},
	{ID: "medium", Kind: "whisper", Name: "Whisper Medium", FileName: "ggml-medium.bin", URL: "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-medium.bin", Size: 1_530_000_000, Description: "準確度較高，但 CPU 處理時間明顯增加。"},
	{ID: "large-v3-turbo-q5", Kind: "whisper", Name: "Whisper Large v3 Turbo Q5", FileName: "ggml-large-v3-turbo-q5_0.bin", URL: "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-large-v3-turbo-q5_0.bin", Size: 574_000_000, Description: "準確度佳，建議 NVIDIA GPU 或高效能 CPU。"},
}

var ollamaCatalog = []CatalogItem{
	{ID: "qwen2.5:1.5b-instruct", Kind: "ollama", Name: "Qwen 2.5 1.5B Instruct", Size: 986_000_000, Description: "速度快、記憶體需求低，適合一般筆電。"},
	{ID: "qwen2.5:3b-instruct", Kind: "ollama", Name: "Qwen 2.5 3B Instruct", Size: 1_900_000_000, Description: "摘要品質與速度較平衡。"},
	{ID: "qwen2.5:7b-instruct", Kind: "ollama", Name: "Qwen 2.5 7B Instruct", Size: 4_700_000_000, Description: "摘要品質較高，建議至少 16 GB 記憶體。"},
}

func Catalog() []CatalogItem {
	items := make([]CatalogItem, 0, len(whisperCatalog)+len(ollamaCatalog))
	items = append(items, whisperCatalog...)
	items = append(items, ollamaCatalog...)
	return items
}

func Find(kind, id string) (CatalogItem, bool) {
	for _, item := range Catalog() {
		if item.Kind == kind && item.ID == id {
			return item, true
		}
	}
	return CatalogItem{}, false
}
