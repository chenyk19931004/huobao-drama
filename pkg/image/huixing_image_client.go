package image

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// HuixingImageClient 汇星云文生图客户端
// 文生图接口: http://azj1.dc.huixingyun.com:55875/webhook/72874db5-0e24-44cb-8f2c-7fa60435e652
// 查询图片接口: http://azj1.dc.huixingyun.com:55875/webhook/fba6d1a8-57af-405f-8752-8f88313d7c10
type HuixingImageClient struct {
	// GenerateEndpoint 文生图接口地址
	GenerateEndpoint string
	// QueryEndpoint 查询图片接口地址
	QueryEndpoint string
	// Port ComfyUI服务端口
	Port string
	// HTTPClient HTTP客户端
	HTTPClient *http.Client
}

// HuixingGenerateResponse 文生图接口返回结果
type HuixingGenerateResponse struct {
	PromptID   string                 `json:"prompt_id"`
	Number     int                    `json:"number"`
	NodeErrors map[string]interface{} `json:"node_errors"`
}

// HuixingQueryResponse 查询图片接口返回结果
type HuixingQueryResponse struct {
	URLs  []string `json:"urls"`
	Count int      `json:"count"`
}

// NewHuixingImageClient 创建汇星云图片客户端
// generateEndpoint: 文生图接口地址
// queryEndpoint: 查询图片接口地址
// port: ComfyUI服务端口，默认8188
func NewHuixingImageClient(generateEndpoint, queryEndpoint, port string) *HuixingImageClient {
	if port == "" {
		port = "8188"
	}
	return &HuixingImageClient{
		GenerateEndpoint: generateEndpoint,
		QueryEndpoint:    queryEndpoint,
		Port:             port,
		HTTPClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

// GenerateImage 提交文生图任务
func (c *HuixingImageClient) GenerateImage(prompt string, opts ...ImageOption) (*ImageResult, error) {
	options := &ImageOptions{
		Width:  720,
		Height: 1080,
	}

	for _, opt := range opts {
		opt(options)
	}

	// 构建请求URL
	// http://azj1.dc.huixingyun.com:55875/webhook/72874db5-0e24-44cb-8f2c-7fa60435e652?prompt=xxx&width=720&high=1080&size=1&port=8188
	params := url.Values{}
	params.Add("prompt", prompt)
	params.Add("width", strconv.Itoa(options.Width))
	params.Add("high", strconv.Itoa(options.Height)) // 注意这里用的是 "high" 而不是 "height"
	params.Add("size", "1")                          // 默认生成1张图片
	params.Add("port", c.Port)

	requestURL := c.GenerateEndpoint + "?" + params.Encode()
	fmt.Printf("[Huixing Image] Request URL: %s\n", requestURL)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	fmt.Printf("[Huixing Image] Response: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析返回结果（返回的是数组）
	var results []HuixingGenerateResponse
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("empty response from Huixing API")
	}

	result := results[0]
	if result.PromptID == "" {
		return nil, fmt.Errorf("no prompt_id in response")
	}

	// 检查是否有节点错误
	if len(result.NodeErrors) > 0 {
		return nil, fmt.Errorf("node errors in response: %v", result.NodeErrors)
	}

	// 返回任务ID，需要后续轮询获取结果
	return &ImageResult{
		TaskID:    result.PromptID,
		Status:    "processing",
		Completed: false,
	}, nil
}

// GetTaskStatus 根据prompt_id查询图片生成结果
func (c *HuixingImageClient) GetTaskStatus(taskID string) (*ImageResult, error) {
	// 构建查询URL
	// http://azj1.dc.huixingyun.com:55875/webhook/fba6d1a8-57af-405f-8752-8f88313d7c10?prompt_id=xxx&port=8188
	params := url.Values{}
	params.Add("prompt_id", taskID)
	params.Add("port", c.Port)

	requestURL := c.QueryEndpoint + "?" + params.Encode()
	fmt.Printf("[Huixing Image] Query URL: %s\n", requestURL)

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	fmt.Printf("[Huixing Image] Query Response: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析返回结果（返回的是数组）
	var results []HuixingQueryResponse
	if err := json.Unmarshal(body, &results); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(results) == 0 {
		// 如果返回空数组，说明图片还在生成中
		return &ImageResult{
			TaskID:    taskID,
			Status:    "processing",
			Completed: false,
		}, nil
	}

	result := results[0]
	if result.Count == 0 || len(result.URLs) == 0 {
		// 没有图片，可能还在生成中
		return &ImageResult{
			TaskID:    taskID,
			Status:    "processing",
			Completed: false,
		}, nil
	}

	// 图片生成完成
	return &ImageResult{
		TaskID:    taskID,
		Status:    "completed",
		ImageURL:  result.URLs[0],
		Completed: true,
	}, nil
}
