package kb

import (
	"encoding/json"
	"fmt"
)

// Song 表示歌曲信息的结构体
// 可以序列化为指定的 JSON 格式
// 包含歌曲的各种元数据

type Song struct {
	Title           string   `json:"title"`
	Artist          string   `json:"artist"`
	Album           string   `json:"album"`
	Duration        int      `json:"duration"`
	ReleaseYear     int      `json:"release_year"`
	Language        string   `json:"language"`
	Theme           string   `json:"theme"`
	Genre           string   `json:"genre"`
	SubGenre        string   `json:"sub_genre"`
	VocalType       string   `json:"vocal_type"`
	MoodTags        []string `json:"mood_tags"`
	SceneTags       []string `json:"scene_tags"`
	LocationTags    []string `json:"location_tags"`
	CulturalTags    []string `json:"cultural_tags"`
	PerformanceForm string   `json:"performance_form"`
}

// ToJSON 将 Song 结构体转换为 JSON 字符串
func (s *Song) ToJSON() (string, error) {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal song to JSON: %w", err)
	}
	return string(data), nil
}

// FromJSON 将 JSON 字符串转换为 Song 结构体
func FromJSON(jsonStr string) (*Song, error) {
	var song Song
	err := json.Unmarshal([]byte(jsonStr), &song)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON to song: %w", err)
	}
	return &song, nil
}