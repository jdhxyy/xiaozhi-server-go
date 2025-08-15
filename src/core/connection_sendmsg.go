package core

import (
	"math/rand"
	"time"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"xiaozhi-server-go/src/core/utils"
)

// sendHelloMessage 发送欢迎消息
func (h *ConnectionHandler) sendHelloMessage() error {
	// 添加安全检查
	if h.conn == nil {
		return fmt.Errorf("连接对象未初始化，无法发送hello消息")
		// 可以添加一个小延迟，防止CPU占用过高
		time.Sleep(100 * time.Millisecond)
	}

	// 其他可能的 nil 检查
	if h.config == nil {
		return fmt.Errorf("配置对象未初始化")
	}

	hello := make(map[string]interface{})
	hello["type"] = "hello"
	hello["version"] = 1
	hello["transport"] = "websocket"
	hello["session_id"] = h.sessionID
	hello["audio_params"] = map[string]interface{}{
		"format":         h.serverAudioFormat,
		"sample_rate":    h.serverAudioSampleRate,
		"channels":       h.serverAudioChannels,
		"frame_duration": h.serverAudioFrameDuration,
	}
	data, err := json.Marshal(hello)
	if err != nil {
		return fmt.Errorf("序列化欢迎消息失败: %v", err)
	}

	return h.conn.WriteMessage(1, data)
}

func (h *ConnectionHandler) sendTTSMessage(state string, text string, textIndex int) error {
	// 发送TTS状态结束通知
	stateMsg := map[string]interface{}{
		"type":        "tts",
		"state":       state,
		"session_id":  h.sessionID,
		"text":        text,
		"index":       textIndex,
		"audio_codec": "opus", // 标识使用Opus编码
	}
	data, err := json.Marshal(stateMsg)
	if err != nil {
		return fmt.Errorf("序列化%s状态失败: %v", state, err)
	}
	if err := h.conn.WriteMessage(1, data); err != nil {
		return fmt.Errorf("发送%s状态失败: %v", state, err)
	}
	return nil
}

func (h *ConnectionHandler) sendSTTMessage(text string) error {
	sttMsg := map[string]interface{}{
		"type":       "stt",
		"text":       text,
		"session_id": h.sessionID,
	}
	jsonData, err := json.Marshal(sttMsg)
	if err != nil {
		return fmt.Errorf("序列化 STT 消息失败: %v", err)
	}
	if err := h.conn.WriteMessage(1, jsonData); err != nil {
		return fmt.Errorf("发送 STT 消息失败: %v", err)
	}

	return nil
}

// sendEmotionMessage 发送情绪消息
func (h *ConnectionHandler) sendEmotionMessage(emotion string) error {
	data := map[string]interface{}{
		"type":       "llm",
		"text":       utils.GetEmotionEmoji(emotion),
		"emotion":    emotion,
		"session_id": h.sessionID,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化情绪消息失败: %v", err)
	}
	return h.conn.WriteMessage(1, jsonData)
}

func (h *ConnectionHandler) sendAudioMessage(filepath string, text string, textIndex int, round int) {
	bFinishSuccess := false
	defer func() {
		// 音频发送完成后，根据配置决定是否删除文件
		h.deleteAudioFileIfNeeded(filepath, "音频发送完成")

		h.LogInfo(fmt.Sprintf("TTS音频发送任务结束(%t): %s, 索引: %d/%d", bFinishSuccess, text, textIndex, h.tts_last_text_index))
		h.providers.asr.ResetStartListenTime()
		if textIndex == h.tts_last_text_index {
			h.sendTTSMessage("stop", "", textIndex)
			if h.closeAfterChat {
				h.Close()
			} else {
				h.clearSpeakStatus()
			}
		}
	}()

	if len(filepath) == 0 {
		return
	}
	// 检查轮次
	if round != h.talkRound {
		h.LogInfo(fmt.Sprintf("sendAudioMessage: 跳过过期轮次的音频: 任务轮次=%d, 当前轮次=%d, 文本=%s",
			round, h.talkRound, text))
		// 即使跳过，也要根据配置删除音频文件
		h.deleteAudioFileIfNeeded(filepath, "跳过过期轮次")
		return
	}

	if atomic.LoadInt32(&h.serverVoiceStop) == 1 { // 服务端语音停止
		h.LogInfo(fmt.Sprintf("sendAudioMessage 服务端语音停止, 不再发送音频数据：%s", text))
		// 服务端语音停止时也要根据配置删除音频文件
		h.deleteAudioFileIfNeeded(filepath, "服务端语音停止")
		return
	}

	var audioData [][]byte
	var duration float64
	var err error

	// 使用TTS提供者的方法将音频转为Opus格式
	if h.serverAudioFormat == "pcm" {
		h.LogInfo("服务端音频格式为PCM，直接发送")
		audioData, duration, err = utils.AudioToPCMData(filepath)
		if err != nil {
			h.LogError(fmt.Sprintf("音频转PCM失败: %v", err))
			return
		}
	} else if h.serverAudioFormat == "opus" {
		audioData, duration, err = utils.AudioToOpusData(filepath)
		if err != nil {
			h.LogError(fmt.Sprintf("音频转Opus失败: %v", err))
			return
		}
	}

	// 发送TTS状态开始通知
	if err := h.sendTTSMessage("sentence_start", text, textIndex); err != nil {
		h.LogError(fmt.Sprintf("发送TTS开始状态失败: %v", err))
		return
	}

	if textIndex == 1 {
		now := time.Now()
		spentTime := now.Sub(h.roundStartTime)
		h.logger.Debug("回复首句耗时 %s 第一句话【%s】, round: %d", spentTime, text, round)
	}
	h.logger.Debug("TTS发送(%s): \"%s\" (索引:%d/%d，时长:%f，帧数:%d)", h.serverAudioFormat, text, textIndex, h.tts_last_text_index, duration, len(audioData))

	// 分时发送音频数据
	if err := h.sendAudioFrames(audioData, text, round); err != nil {
		h.LogError(fmt.Sprintf("分时发送音频数据失败: %v", err))
		return
	}

	// 发送TTS状态结束通知
	if err := h.sendTTSMessage("sentence_end", text, textIndex); err != nil {
		h.LogError(fmt.Sprintf("发送TTS结束状态失败: %v", err))
		return
	}

	bFinishSuccess = true
}

func (h *ConnectionHandler) sendMusic(songFilepaths []string, texts []string, textIndex int, round int) {
	// 初始化随机数生成器
	rand.Seed(time.Now().UnixNano())
	bFinishSuccess := false
	defer func() {
		h.LogInfo(fmt.Sprintf("music音频发送任务结束(%t): 索引: %d/%d", bFinishSuccess, textIndex, h.tts_last_text_index))
		h.providers.asr.ResetStartListenTime()
		if textIndex == h.tts_last_text_index {
			h.sendTTSMessage("stop", "", textIndex)
			if h.closeAfterChat {
				h.Close()
			} else {
				h.clearSpeakStatus()
			}
		}
	}()

	// 检查轮次
	if round != h.talkRound {
		h.LogInfo(fmt.Sprintf("sendMusic: 跳过过期轮次的音频: 任务轮次=%d, 当前轮次=%d",
			round, h.talkRound))
		return
	}

	if atomic.LoadInt32(&h.serverVoiceStop) == 1 { // 服务端语音停止
		h.LogInfo(fmt.Sprintf("sendMusic 服务端语音停止, 不再发送音频数据"))
		return
	}

	var audioData [][]byte
	var duration float64
	var err error

	// 检查播放列表是否为空
	if len(songFilepaths) == 0 {
		h.LogError("播放列表为空，无法播放音乐")
		return
	}

	// 初始化变量
	isFirstLoop := true
	currentIndex := 0
	lastPlayedIndex := -1

	// 无限循环播放音乐
	for {
		// 检查是否需要停止播放
		if atomic.LoadInt32(&h.serverVoiceStop) == 1 {
			h.LogInfo("音乐播放已停止")
			return
		}

		// 检查轮次是否已变更
		if round != h.talkRound {
			h.LogInfo(fmt.Sprintf("sendMusic: 跳过过期轮次的音频: 任务轮次=%d, 当前轮次=%d", round, h.talkRound))
			return
		}

		var randIndex int
		// 第一次循环按顺序播放
		if isFirstLoop {
			randIndex = currentIndex
			currentIndex++

			// 检查是否完成第一次循环
			if currentIndex >= len(songFilepaths) {
				isFirstLoop = false
				lastPlayedIndex = randIndex // 记录最后播放的歌曲索引
				currentIndex = 0
			}
		} else {
			// 随机播放，确保不与上一首相同
			for {
				randIndex = rand.Intn(len(songFilepaths))
				if randIndex != lastPlayedIndex {
					break
				}
			}

			lastPlayedIndex = randIndex
		}

		songFilepath := songFilepaths[randIndex]
		text := texts[randIndex]

		// 使用TTS提供者的方法将音频转为Opus格式
		audioData, duration, err = getAudioData(songFilepath, h)
		if err != nil {
			h.LogError(fmt.Sprintf("获取音频数据失败: %v", err))
			continue // 出错时跳过当前歌曲，继续播放下一首
		}

		// 发送TTS状态开始通知
		if err := h.sendTTSMessage("sentence_start", text, textIndex); err != nil {
			h.LogError(fmt.Sprintf("发送TTS开始状态失败: %v", err))
			continue
		}

		if textIndex == 1 {
			now := time.Now()
			spentTime := now.Sub(h.roundStartTime)
			h.logger.Debug("回复首句耗时 %s 第一句话【%s】, round: %d", spentTime, text, round)
		}
		h.logger.Debug("音乐播放(%s): \"%s\" (索引:%d/%d，时长:%f，帧数:%d)", h.serverAudioFormat, text, textIndex, h.tts_last_text_index, duration, len(audioData))

		// 分时发送音频数据
		if err := h.sendAudioFrames(audioData, text, round); err != nil {
			h.LogError(fmt.Sprintf("分时发送音频数据失败: %v", err))
			continue
		}

		// 发送TTS状态结束通知
		if err := h.sendTTSMessage("sentence_end", text, textIndex); err != nil {
			h.LogError(fmt.Sprintf("发送TTS结束状态失败: %v", err))
			continue
		}
	}

	bFinishSuccess = true
}

func getAudioData(songFilepath string, h *ConnectionHandler) (audioData [][]byte, duration float64, err error) {
	// 提取文件名（不含扩展名）
	filename := filepath.Base(songFilepath)
	filenameWithoutExt := strings.TrimSuffix(filename, filepath.Ext(filename))

	// 构造opus文件路径
	opusDir := filepath.Join("", "music-opus")
	opusFilePath := filepath.Join(opusDir, filenameWithoutExt+"_.opus")
	durationFilePath := filepath.Join(opusDir, filenameWithoutExt+"_.duration")

	// 检查opus文件是否存在
	if _, err := os.Stat(opusFilePath); os.IsNotExist(err) {
		// 文件不存在，调用转换函数
		audioData, duration, err = utils.AudioToOpusData(songFilepath)
		if err != nil {
			h.LogError(fmt.Sprintf("音频转Opus失败: %v", err))
			return nil, 0, err
		}

		// 确保music-opus目录存在
		if err := os.MkdirAll(opusDir, 0755); err != nil {
			h.LogError(fmt.Sprintf("创建opus目录失败: %v", err))
		} else {
			// 保存全部opus数据到文件（带长度信息）
			for i, data := range audioData {
				// 准备包含长度信息的数据
				lengthBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(lengthBytes, uint32(len(data)))
				dataWithLength := append(lengthBytes, data...)

				if i == 0 {
					// 第一次写入，创建或覆盖文件
					if err := utils.SaveAudioFile(dataWithLength, opusFilePath); err != nil {
						h.LogError(fmt.Sprintf("保存Opus文件失败: %v", err))
					}
				} else {
					// 后续写入，追加到文件
					if err := utils.AppendAudioFile(dataWithLength, opusFilePath); err != nil {
						h.LogError(fmt.Sprintf("追加Opus文件失败: %v", err))
					}
				}
			}

			// 保存duration到文件
			durationFile, err := os.Create(durationFilePath)
			if err != nil {
				h.LogError(fmt.Sprintf("创建duration文件失败: %v", err))
			} else {
				fmt.Fprintf(durationFile, "%f", duration)
				durationFile.Close()
			}
		}
	} else {
		// 文件存在，直接读取
		opusData, err := os.ReadFile(opusFilePath)
		if err != nil {
			h.LogError(fmt.Sprintf("读取Opus文件失败: %v", err))
			return nil, 0, err
		}

		// 从文件中读取带长度信息的opus数据
		audioData = [][]byte{}
		offset := 0
		dataLen := len(opusData)

		for offset < dataLen {
			// 读取长度信息（4字节）
			if offset+4 > dataLen {
				h.LogError("Opus文件格式错误：缺少足够的长度信息")
				break
			}

			length := int(binary.LittleEndian.Uint32(opusData[offset:offset+4]))
			offset += 4

			// 读取数据
			if offset+length > dataLen {
				h.LogError("Opus文件格式错误：数据长度不足")
				break
			}

			data := opusData[offset:offset+length]
			audioData = append(audioData, data)
			offset += length
		}

		// 如果解析后的帧数量为0，则使用原始数据作为备选
		if len(audioData) == 0 {
			audioData = [][]byte{opusData}
		}

		// 读取duration
		durationData, err := os.ReadFile(durationFilePath)
		if err != nil {
			h.LogError(fmt.Sprintf("读取duration文件失败: %v", err))
			return nil, 0, err
		}
		fmt.Sscanf(string(durationData), "%f", &duration)
	}

	return audioData, duration, nil
}

// sendAudioFrames 分时发送音频帧，避免撑爆客户端缓冲区
func (h *ConnectionHandler) sendAudioFrames(audioData [][]byte, text string, round int) error {
	if len(audioData) == 0 {
		return nil
	}

	startTime := time.Now()
	playPosition := 0 // 播放位置（毫秒）

	// 预缓冲：发送前几帧，提升播放流畅度
	preBufferFrames := 3
	if len(audioData) < preBufferFrames {
		preBufferFrames = len(audioData)
	}
	preBufferTime := time.Duration(h.serverAudioFrameDuration*preBufferFrames) * time.Millisecond // 预缓冲时间（毫秒）

	// 发送预缓冲帧
	for i := 0; i < preBufferFrames; i++ {
		// 检查是否被打断
		if atomic.LoadInt32(&h.serverVoiceStop) == 1 || round != h.talkRound {
			h.LogInfo(fmt.Sprintf("音频发送被中断(预缓冲阶段): 帧=%d/%d, 文本=%s", i+1, preBufferFrames, text))
			return nil
		}

		if err := h.conn.WriteMessage(2, audioData[i]); err != nil {
			return fmt.Errorf("发送预缓冲音频帧失败: %v", err)
		}
		playPosition += h.serverAudioFrameDuration
	}

	// 发送剩余音频帧
	remainingFrames := audioData[preBufferFrames:]
	for i, chunk := range remainingFrames {
		// 检查是否被打断或轮次变化
		if atomic.LoadInt32(&h.serverVoiceStop) == 1 || round != h.talkRound {
			h.LogInfo(fmt.Sprintf("音频发送被中断: 帧=%d/%d, 文本=%s", i+preBufferFrames+1, len(audioData), text))
			return nil
		}

		// 检查连接是否关闭
		select {
		case <-h.stopChan:
			return nil
		default:
		}

		// 计算预期发送时间
		expectedTime := startTime.Add(time.Duration(playPosition)*time.Millisecond - preBufferTime)
		currentTime := time.Now()
		delay := expectedTime.Sub(currentTime)

		// 流控延迟处理
		if delay > 0 {
			// 使用简单的可中断睡眠
			ticker := time.NewTicker(10 * time.Millisecond) // 固定10ms检查间隔
			defer ticker.Stop()

			endTime := time.Now().Add(delay)
			for time.Now().Before(endTime) {
				select {
				case <-ticker.C:
					// 检查中断条件
					if atomic.LoadInt32(&h.serverVoiceStop) == 1 || round != h.talkRound {
						h.LogInfo(fmt.Sprintf("音频发送在延迟中被中断: 帧=%d/%d, 文本=%s", i+preBufferFrames+1, len(audioData), text))
						return nil
					}
				case <-h.stopChan:
					return nil
				}
			}
		}

		// 发送音频帧
		if err := h.conn.WriteMessage(2, chunk); err != nil {
			return fmt.Errorf("发送音频帧失败: %v", err)
		}

		playPosition += h.serverAudioFrameDuration
	}
	time.Sleep(preBufferTime) // 确保预缓冲时间已过
	spentTime := time.Since(startTime).Milliseconds()
	h.LogInfo(fmt.Sprintf("音频帧发送完成: 总帧数=%d, 总时长=%dms, 总耗时:%dms 文本=%s", len(audioData), playPosition, spentTime, text))
	return nil
}
