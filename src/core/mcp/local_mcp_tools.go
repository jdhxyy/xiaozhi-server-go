package mcp

import (
	"context"
	"strings"
	"time"
	"xiaozhi-server-go/src/core/types"
)

func (c *LocalClient) AddToolExit() error {

	InputSchema := ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"say_goodbye": map[string]any{
				"type":        "string",
				"description": "用户友好结束对话的告别语",
			},
		},
		Required: []string{"say_goodbye"},
	}

	c.AddTool("exit",
		"当用户想结束对话或需要退出系统时调用",
		InputSchema,
		func(ctx context.Context, args map[string]any) (interface{}, error) {
			c.logger.Info("用户请求退出对话，告别语：%s", args["say_goodbye"])
			res := types.ActionResponse{
				Action: types.ActionTypeCallHandler, // 动作类型
				Result: types.ActionResponseCall{
					FuncName: "mcp_handler_exit",  // 函数名
					Args:     args["say_goodbye"], // 函数参数
				},
			}
			return res, nil
		})

	return nil
}

func (c *LocalClient) AddToolTime() error {

	InputSchema := ToolInputSchema{
		Type:       "object",
		Properties: map[string]any{},
		Required:   []string{},
	}

	c.AddTool("get_time",
		"获取今天日期或者当前时间信息时调用",
		InputSchema,
		func(ctx context.Context, args map[string]any) (interface{}, error) {
			now := time.Now()
			time := now.Format("2006-01-02 15点04分05秒")
			week := now.Weekday().String()
			str := "当前时间是 " + time + "，今天是" + week + "。"
			res := types.ActionResponse{
				Action: types.ActionTypeReqLLM, // 动作类型
				Result: str,                    // 函数参数
			}
			return res, nil
		})

	return nil
}

func (c *LocalClient) AddToolChangeRole() error {
	roles := c.cfg.Roles
	prompts := map[string]string{}
	roleNames := ""
	if roles == nil {
		c.logger.Warn("AddToolChangeRole: roles settings is nil or empty, Skipping tool registration")
		return nil
	} else {
		for _, role := range roles {
			items := strings.Split(role, "@")
			prompts[items[0]] = items[1]
			roleNames += items[0] + ", "
		}
	}

	InputSchema := ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"role": map[string]any{
				"type":        "string",
				"description": "新的角色名称",
			},
		},
		Required: []string{"role"},
	}

	c.AddTool("change_role",
		"当用户想切换角色/模型性格/助手名字时调用,可选的角色有：["+roleNames+"]",
		InputSchema,
		func(ctx context.Context, args map[string]any) (interface{}, error) {
			role := args["role"].(string)
			res := types.ActionResponse{
				Action: types.ActionTypeCallHandler, // 动作类型
				Result: types.ActionResponseCall{
					FuncName: "mcp_handler_change_role", // 函数名
					Args: map[string]string{
						"role":   role, // 函数参数
						"prompt": prompts[role],
					},
				},
			}
			return res, nil
		})

	return nil
}

func (c *LocalClient) AddToolChangeVoice() error {

	voices := []string{}
	if ttsType, ok := c.cfg.SelectedModule["TTS"]; ok && ttsType != "" {
		voices = c.cfg.TTS[ttsType].SurportedVoices
	}
	voiceDes := strings.Join(voices, ", ")

	InputSchema := ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"voice": map[string]any{
				"type":        "string",
				"description": "新的语音名称，音色描述中的第一部分",
			},
		},
		Required: []string{"voice"},
	}

	c.AddTool("change_voice",
		"当用户想要更换角色语音或音色时调用，当前支持的音色有: "+voiceDes,
		InputSchema,
		func(ctx context.Context, args map[string]any) (interface{}, error) {
			voice := args["voice"].(string)
			res := types.ActionResponse{
				Action: types.ActionTypeCallHandler, // 动作类型
				Result: types.ActionResponseCall{
					FuncName: "mcp_handler_change_voice", // 函数名
					Args:     voice,                      // 函数参数
				},
			}
			return res, nil
		})

	return nil
}

func (c *LocalClient) AddToolPlayMusic() error {
	InputSchema := ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"song_requirement": map[string]any{
				"type":        "string",
				"description": "歌曲要求，可包含具体歌曲名、歌手、音乐风格、场景、心情、乐器等信息。示例: ```用户:播放周杰伦的歌曲 参数：周杰伦的歌曲``` ```用户:播放适合放松的钢琴音乐 参数：放松的钢琴曲```",
			},
		},
		Required: []string{"song_requirement"},
	}

	c.AddTool("play_music",
		"音乐播放工具。当用户想要播放指定歌曲时调用",
		InputSchema,
		func(ctx context.Context, args map[string]any) (interface{}, error) {
			song_requirement := args["song_requirement"].(string)
			res := types.ActionResponse{
				Action: types.ActionTypeCallHandler, // 动作类型
				Result: types.ActionResponseCall{
					FuncName: "mcp_handler_play_music", // 函数名
					Args:     song_requirement,                // 函数参数
				},
			}
			return res, nil
		})
	
	InputSchemaSearch := ToolInputSchema{
		Type: "object",
		Properties: map[string]any{
			"song_requirement": map[string]any{
				"type":        "string",
				"description": "歌曲要求，可包含具体歌曲名、歌手、音乐风格、场景、心情、乐器等信息。示例: ```用户:播放周杰伦的歌曲 参数：周杰伦的歌曲``` ```用户:播放适合放松的钢琴音乐 参数：放松的钢琴曲```",
			},
		},
		Required: []string{"song_requirement"},
	}

	c.AddTool("search_music",
		"音乐库搜索工具。当用户想要推荐歌曲/搜索歌曲时调用，会根据用户的要求搜索音乐库，返回符合要求的歌曲列表",

		InputSchemaSearch,
		func(ctx context.Context, args map[string]any) (interface{}, error) {
			song_requirement := args["song_requirement"].(string)
			res := types.ActionResponse{
				Action: types.ActionTypeCallHandler, // 动作类型
				Result: types.ActionResponseCall{
					FuncName: "mcp_handler_search_music", // 函数名
					Args:     song_requirement,           // 函数参数
				},
			}
			return res, nil
		})

	return nil
}
