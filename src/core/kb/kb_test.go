package kb

import (
	"fmt"
	"testing"
)

func TestCase1(t *testing.T) {
	// 创建一个JSON字符串
	songJSON := `{
		"title": "青花瓷-周杰伦",
		"artist": "周杰伦",
		"album": "我很忙",
		"duration": 282,
		"release_year": 2007,
		"language": "中文",
		"theme": "中国风、爱情",
		"genre": "流行",
		"style": "中国风",
		"vocal_type": "独唱",
		"mood_tags": ["古典", "深情", "优雅"],
		"scenario_tags": ["古风活动", "安静聆听", "文化鉴赏"],
		"cultural_tags": ["中国传统文化", "陶瓷艺术"]
	}`

	// 调用Add函数
	err := Add(songJSON)
	if err != nil {
		fmt.Printf("添加文档失败: %v\n", err)
		return
	}

	fmt.Println("文档添加成功！")

	songJSON2 := `{
		"title": "十年-陈奕迅",
		"artist": "陈奕迅",
		"album": "黑·白·灰",
		"duration": 295,
		"release_year": 2003,
		"language": "中文",
		"theme": "爱情、回忆",
		"genre": "流行",
		"style": "抒情流行",
		"vocal_type": "独唱",
		"mood_tags": ["深情", "怀旧", "伤感"],
		"scenario_tags": ["离别场景", "回忆时刻", "深夜聆听"],
		"cultural_tags": ["都市情感", "当代流行文化"]
	}`

	// 调用Add函数
	err = Add(songJSON2)
	if err != nil {
		fmt.Printf("添加文档失败: %v\n", err)
		return
	}

	fmt.Println("文档添加成功！")

	docs, err := Search("中国风", 2)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range docs {
		fmt.Println(v)
	}
}


func TestCase2(t *testing.T) {
	if IsSongExist("歌曲标题是 十年") == true {
		fmt.Println("歌曲已存在")
		return
	}

	// 创建一个JSON字符串
	songJSON := `{
		"title": "十年-陈奕迅",
		"artist": "陈奕迅",
		"album": "黑·白·灰",
		"duration": 295,
		"release_year": 2003,
		"language": "中文",
		"theme": "爱情、回忆",
		"genre": "流行",
		"style": "抒情流行",
		"vocal_type": "独唱",
		"mood_tags": ["深情", "怀旧", "伤感"],
		"scenario_tags": ["离别场景", "回忆时刻", "深夜聆听"],
		"cultural_tags": ["都市情感", "当代流行文化"]
	}`

	// 调用Add函数
	err := Add(songJSON)
	if err != nil {
		fmt.Printf("添加文档失败: %v\n", err)
		return
	}

	fmt.Println("文档添加成功！")
}
