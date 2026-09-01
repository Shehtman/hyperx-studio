// Package webui содержит интерфейс, вшитый прямо в исполняемый файл.
// Благодаря этому программа остаётся одним файлом без внешних ресурсов.
package webui

import "embed"

//go:embed index.html style.css app.js icon.svg
var Files embed.FS
