package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

var logEntry *widget.Entry
var progress *widget.ProgressBar

func main() {
	myApp := app.New()
	window := myApp.NewWindow("ZIP伪加密修复工具 v1.2")
	window.Resize(fyne.NewSize(800, 500))
	// 加载图标资源
	icon := fyne.NewStaticResource("icon", resourceIconPng.StaticContent)

	// 设置窗口或应用图标
	window.SetIcon(icon)
	myApp.SetIcon(icon)
	// GUI组件初始化
	fileEntry := widget.NewEntry()
	fileEntry.SetPlaceHolder("请选择ZIP文件路径")

	logEntry = widget.NewEntry()
	logEntry.MultiLine = true
	logEntry.Wrapping = fyne.TextWrapBreak
	// logEntry.Disable()
	logScroll := container.NewScroll(logEntry)
	logScroll.SetMinSize(fyne.NewSize(780, 200)) // 设置最小高度

	// 文件选择按钮
	fileBtn := widget.NewButtonWithIcon("", theme.FolderIcon(), func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err == nil && reader != nil {
				fileEntry.SetText(reader.URI().Path())
			}
		}, window)
	})
	fileBtn.Importance = widget.LowImportance

	// 修复按钮
	fixBtn := widget.NewButtonWithIcon("开始修复", theme.ConfirmIcon(), func() {
		progress.Show()
		fileBtn.Disable()
		go func() {
			defer func() {
				progress.Hide()
				fileBtn.Enable()
			}()
			err := FixPseudoEncryption(fileEntry.Text)
			if err != nil {
				updateLog("修复失败: " + err.Error())
			} else {
				updateLog(fmt.Sprintf("已修复文件: %s\n请尝试用压缩软件解压",
					filepath.Base(fileEntry.Text)))
			}
		}()
	})
	fixBtn.Importance = widget.HighImportance

	// 进度条
	progress = widget.NewProgressBar()
	progress.Hide()

	// 布局构建
	header := container.NewBorder(
		nil,
		nil,
		fileBtn,
		fixBtn,
		container.NewMax(fileEntry),
	)

	mainContent := container.NewBorder(
		container.NewVBox( // 顶部区域
			header,
			widget.NewSeparator(),
			progress,
		),
		nil, nil, nil, // 没有底部、左、右边框
		container.NewMax( // 中间区域自动扩展
			container.NewBorder(
				widget.NewLabel("操作日志:"),
				nil, nil, nil,
				logScroll,
			),
		),
	)

	window.SetContent(mainContent)
	window.ShowAndRun()
}

// 伪加密修复核心函数[7,8]
func FixPseudoEncryption(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("文件读取失败: %v", err)
	}

	modified := false
	originalData := make([]byte, len(data))
	copy(originalData, data)

	// 修复数据区加密标志（查找PK0304）
	dataSign := []byte{0x50, 0x4B, 0x03, 0x04}
	for offset := 0; ; {
		idx := bytes.Index(data[offset:], dataSign)
		if idx == -1 {
			break
		}
		pos := offset + idx
		if pos+6 < len(data) && data[pos+6]%2 == 1 {
			data[pos+6] = 0x00 // 第6字节改为偶数[8]
			modified = true
		}
		offset = pos + 4
	}

	// 修复目录区加密标志（查找PK0102）[7]
	dirSign := []byte{0x50, 0x4B, 0x01, 0x02}
	for offset := 0; ; {
		idx := bytes.Index(data[offset:], dirSign)
		if idx == -1 {
			break
		}
		pos := offset + idx
		if pos+8 < len(data) && data[pos+8]%2 == 1 {
			data[pos+8] = 0x00 // 第8字节改为偶数[6]
			modified = true
		}
		offset = pos + 4
	}

	if !modified {
		return errors.New("未检测到伪加密特征")
	}

	// 验证修复结果
	if bytes.Equal(data, originalData) {
		return errors.New("文件未发生修改")
	}

	// 创建备份文件
	backupPath := path + ".bak"
	if err := os.WriteFile(backupPath, originalData, 0644); err != nil {
		return fmt.Errorf("备份文件创建失败: %v", err)
	}

	// 写入修复后的文件
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("文件写入失败: %v", err)
	}

	return nil
}

func updateLog(text string) {
	logEntry.SetText(logEntry.Text + text + "\n")
}
