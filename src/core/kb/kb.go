package kb

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/philippgille/chromem-go"
	"runtime"
)

const (
	// KBPath = "../../../music-kb"
	KBPath         = "music-kb"
	CollectionName = "song"
	SimilarityThreshold = 0.6
)

var gCtx context.Context
var gDB *chromem.DB
var gCollection *chromem.Collection

func init() {
	gCtx = context.Background()

	var err error
	gDB, err = chromem.NewPersistentDB(KBPath, false)
	// gDB = chromem.NewDB()
	if err != nil {
		panic(err)
	}

	// 配置Ollama嵌入函数
	ollamaURL := "http://localhost:11434/api"
	embeddingModelName := "bge-m3:latest" // 嵌入模型
	embeddingFunc := chromem.NewEmbeddingFuncOllama(embeddingModelName, ollamaURL)

	// 创建集合
	gCollection = gDB.GetCollection(CollectionName, embeddingFunc)
	if gCollection == nil {
		gCollection, err = gDB.CreateCollection(CollectionName, nil, embeddingFunc)
		if err != nil {
			panic(err)
		}
	}
}

// Add 向知识库添加文档
func Add(jsonStr string) error {
	// 生成UUID作为文档ID
	id := uuid.New().String()

	// 添加文档
	err := gCollection.AddDocuments(gCtx, []chromem.Document{
		{
			ID:      id,
			Content: jsonStr,
		},
	}, runtime.NumCPU())

	return err
}

// Search 搜索知识库
func Search(query string, nResults int) ([]Song, error) {
	res, err := gCollection.Query(gCtx, query, nResults, nil, nil)
	if err != nil {
		return nil, err
	}

	var docs []Song
	for _, v := range res {
		// if v.Similarity < SimilarityThreshold {
		// 	continue
		// }

		s, err := FromJSON(v.Content)
		if err != nil {
			continue
		}
		docs = append(docs, *s)
	}
	return docs, nil
}

func IsSongExist(song string) bool {
	res, err := gCollection.Query(gCtx, song, 1, nil, nil)
	if err != nil {
		return false
	}
	fmt.Println("相似度:", res[0].Similarity)
	return res[0].Similarity > SimilarityThreshold
}
