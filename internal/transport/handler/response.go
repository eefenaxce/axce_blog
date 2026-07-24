package handler

import "github.com/gofiber/fiber/v3"

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

func Success(c fiber.Ctx, data interface{}, msg string) error {
	return c.JSON(Response{
		Code: fiber.StatusOK,
		Data: data,
		Msg:  msg,
	})
}

func Created(c fiber.Ctx, data interface{}, msg string) error {
	return c.Status(fiber.StatusCreated).JSON(Response{
		Code: fiber.StatusCreated,
		Data: data,
		Msg:  msg,
	})
}

func Error(c fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(Response{
		Code: status,
		Data: nil,
		Msg:  msg,
	})
}