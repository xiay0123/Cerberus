// Package response 提供 Cerberus 服务的统一 API 响应格式。
//
// 所有 API 接口返回统一的 JSON 结构，便于客户端解析和处理：
//
//	{
//	    "code": 0,           // 业务状态码，0 表示成功
//	    "message": "ok",     // 响应消息
//	    "data": {...}        // 响应数据（可选）
//	}
//
// 使用方式：
//
//	response.OK(c, data)        // 返回成功响应
//	response.Created(c, data)   // 返回资源创建成功响应
//	response.Error(c, 400, "参数错误")  // 返回错误响应
package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIResponse 定义统一的 API 响应结构体。
// 所有接口返回的 JSON 都遵循此格式，确保客户端可以统一处理。
type APIResponse struct {
	// Code 是业务状态码。
	// 0 表示成功，非 0 表示业务错误。
	// 注意：与 HTTP 状态码不同，这是应用层的状态码。
	Code int `json:"code"`

	// Message 是响应消息，用于描述请求处理结果。
	// 成功时通常为 "ok" 或 "created"，失败时为错误描述。
	Message string `json:"message"`

	// Data 是响应数据，可以是任意类型。
	// 成功时包含业务数据，失败时通常为空。
	// 使用 omitempty 标签，当 data 为 nil 时不返回该字段。
	Data interface{} `json:"data,omitempty"`
}

// listResponse 用于 Swagger 文档的分页响应结构。
type listResponse struct {
	Items interface{} `json:"items"` // 数据列表
	Total int64       `json:"total"` // 总数量
	Page  int         `json:"page"`  // 当前页码
	Size  int         `json:"size"`  // 每页数量
}

// OK 返回 HTTP 200 成功响应。
//
// 参数：
//   - c: Gin 上下文，用于写入 HTTP 响应
//   - data: 响应数据，可以是任意类型（结构体、map、切片等）
//
// 示例：
//
//	response.OK(c, gin.H{"total": 100})
//	response.OK(c, license)
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Code:    0,
		Message: "ok",
		Data:    data,
	})
}

// Created 返回 HTTP 201 资源创建成功响应。
//
// 用于 POST 请求创建资源成功的场景，HTTP 状态码为 201 Created。
//
// 参数：
//   - c: Gin 上下文
//   - data: 新创建的资源数据
//
// 示例：
//
//	response.Created(c, newLicense)
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Code:    0,
		Message: "created",
		Data:    data,
	})
}

// Error 返回错误响应。
//
// 根据传入的 HTTP 状态码返回相应的错误信息。
// 业务状态码 Code 与 HTTP 状态码相同，便于客户端统一处理。
//
// 参数：
//   - c: Gin 上下文
//   - status: HTTP 状态码（如 400、401、403、404、500 等）
//   - msg: 错误消息，描述错误原因
//
// 示例：
//
//	response.Error(c, 400, "参数错误")
//	response.Error(c, 404, "License not found")
//	response.Error(c, 500, "内部服务器错误")
func Error(c *gin.Context, status int, msg string) {
	c.JSON(status, APIResponse{
		Code:    status,
		Message: msg,
	})
}
