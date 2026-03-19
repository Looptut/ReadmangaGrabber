package mangalib

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/cavaliergopher/grab/v3"
	"github.com/goware/urlx"

	"github.com/lirix360/ReadmangaGrabber/config"
	"github.com/lirix360/ReadmangaGrabber/data"
	"github.com/lirix360/ReadmangaGrabber/history"
	"github.com/lirix360/ReadmangaGrabber/pdf"
	"github.com/lirix360/ReadmangaGrabber/tools"
)

const apiBase = "https://api.cdnlibs.org/api/manga"
const imgBase = "https://img3.mixlib.me"

var (
	ErrNoToken    = errors.New("токен MangaLib не найден — добавьте его в Настройки")
	ErrBadToken   = errors.New("токен MangaLib недействителен или истёк — обновите его в Настройках")
	ErrNotFound   = errors.New("манга не найдена (404) — проверьте правильность адреса")
	ErrServerHTML = errors.New("сервер вернул HTML вместо JSON — возможно, блокировка DDoS-Guard")
	ErrEmptyPages = errors.New("список страниц главы пуст — попробуйте позже")
)

type chaptersResponse struct {
	Data []struct {
		Volume string `json:"volume"`
		Number string `json:"number"`
		Name   string `json:"name"`
	} `json:"data"`
}

type chapterResponse struct {
	Data struct {
		Pages []struct {
			URL string `json:"url"`
		} `json:"pages"`
	} `json:"data"`
}

func getAuthToken() string {
	data, err := os.ReadFile("mangalib_token.txt")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func apiRequest(url string) ([]byte, error) {
	token := getAuthToken()
	if token == "" {
		slog.Error("Токен MangaLib не найден", slog.String("url", url))
		return nil, ErrNoToken
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("Ошибка при создании запроса", slog.String("url", url), slog.String("Message", err.Error()))
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("site-id", "1")
	req.Header.Set("User-Agent", config.Cfg.UserAgent)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("client-time-zone", "Asia/Novosibirsk")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Referer", "https://mangalib.me/")
	req.Header.Set("Origin", "https://mangalib.me")

	slog.Info("API запрос MangaLib", slog.String("url", url), slog.Int("token_len", len(token)))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("Ошибка при выполнении запроса", slog.String("url", url), slog.String("Message", err.Error()))
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("Ошибка при чтении ответа", slog.String("url", url), slog.String("Message", err.Error()))
		return nil, err
	}

	slog.Info("API ответ MangaLib",
		slog.String("url", url),
		slog.Int("status", resp.StatusCode),
		slog.String("body_start", truncate(string(body), 150)),
	)

	switch resp.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusForbidden:
		slog.Error("Доступ запрещён (403) — токен недействителен", slog.String("url", url))
		return nil, ErrBadToken
	case http.StatusUnauthorized:
		slog.Error("Не авторизован (401) — токен недействителен", slog.String("url", url))
		return nil, ErrBadToken
	case http.StatusNotFound:
		slog.Error("Не найдено (404)", slog.String("url", url))
		return nil, ErrNotFound
	default:
		return nil, fmt.Errorf("HTTP %d при запросе %s", resp.StatusCode, url)
	}

	bodyStr := strings.TrimSpace(string(body))
	if strings.HasPrefix(bodyStr, "<") {
		slog.Error("Получен HTML вместо JSON",
			slog.String("url", url),
			slog.String("body_start", truncate(bodyStr, 200)),
		)
		return nil, ErrServerHTML
	}

	return body, nil
}

func getMangaSlug(mangaURL string) string {
	u, _ := urlx.Parse(mangaURL)
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	return parts[len(parts)-1]
}

func GetMangaInfo(mangaURL string) (data.MangaInfo, error) {
	return data.MangaInfo{}, nil
}

func GetChaptersList(mangaURL string) ([]data.ChaptersList, error) {
	slug := getMangaSlug(mangaURL)
	url := fmt.Sprintf("%s/%s/chapters", apiBase, slug)

	body, err := apiRequest(url)
	if err != nil {
		return nil, err
	}

	var resp chaptersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		slog.Error("Ошибка при разборе списка глав",
			slog.String("Message", err.Error()),
			slog.String("body_start", truncate(string(body), 200)),
		)
		return nil, fmt.Errorf("ошибка разбора ответа: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, errors.New("список глав пуст — проверьте адрес манги")
	}

	var chaptersList []data.ChaptersList
	for _, ch := range resp.Data {
		title := fmt.Sprintf("Том %s Глава %s", ch.Volume, ch.Number)
		if ch.Name != "" {
			title += " " + ch.Name
		}
		chaptersList = append(chaptersList, data.ChaptersList{
			Title: title,
			Path:  "v" + ch.Volume + "/c" + ch.Number,
		})
	}

	slog.Info("Список глав получен", slog.String("manga", mangaURL), slog.Int("count", len(chaptersList)))
	return chaptersList, nil
}

func DownloadManga(downData data.DownloadOpts) error {
	slug := getMangaSlug(downData.MangaURL)

	var chaptersList []data.ChaptersList
	var err error
	var saveChapters []string
	savedFilesByVol := make(map[string][]string)

	switch downData.Type {
	case "all":
		chaptersList, err = GetChaptersList(downData.MangaURL)
		if err != nil {
			slog.Error("Ошибка при получении списка глав", slog.String("Message", err.Error()))
			data.WSChan <- data.WSData{
				Cmd: "updateLog",
				Payload: map[string]interface{}{"type": "err", "text": "Ошибка: " + err.Error()},
			}
			return err
		}
		time.Sleep(1 * time.Second)
	case "chapters":
		chaptersRaw := strings.Split(strings.Trim(downData.Chapters, "[] \""), "\",\"")
		for _, ch := range chaptersRaw {
			chaptersList = append(chaptersList, data.ChaptersList{Path: ch})
		}
	}

	chaptersTotal := len(chaptersList)
	chaptersCur := 0

	data.WSChan <- data.WSData{
		Cmd: "initProgress",
		Payload: map[string]interface{}{"valNow": 0, "valMax": chaptersTotal, "width": 0},
	}

	for _, chapter := range chaptersList {
		parts := strings.Split(chapter.Path, "/")
		volume := strings.TrimLeft(parts[0], "v")

		chSavedFiles, err := DownloadChapter(downData, chapter, slug)
		if err != nil {
			data.WSChan <- data.WSData{
				Cmd: "updateLog",
				Payload: map[string]interface{}{
					"type": "err",
					"text": "-- Ошибка при скачивании главы " + chapter.Path + ": " + err.Error(),
				},
			}
			if errors.Is(err, ErrNoToken) || errors.Is(err, ErrBadToken) {
				return err
			}
		}

		savedFilesByVol[volume] = append(savedFilesByVol[volume], chSavedFiles...)
		chaptersCur++
		saveChapters = append(saveChapters, chapter.Path)

		time.Sleep(time.Duration(config.Cfg.Mangalib.TimeoutChapter) * time.Millisecond)

		data.WSChan <- data.WSData{
			Cmd: "updateProgress",
			Payload: map[string]interface{}{
				"valNow": chaptersCur,
				"width":  tools.GetPercent(chaptersCur, chaptersTotal),
			},
		}
	}

	chapterPath := path.Join(config.Cfg.Savepath, downData.SavePath)

	if downData.PDFvol == "1" {
		data.WSChan <- data.WSData{Cmd: "updateLog", Payload: map[string]interface{}{"type": "std", "text": "Создаю PDF для томов"}}
		pdf.CreateVolPDF(chapterPath, savedFilesByVol, downData.Del)
	}
	if downData.PDFall == "1" {
		data.WSChan <- data.WSData{Cmd: "updateLog", Payload: map[string]interface{}{"type": "std", "text": "Создаю PDF для манги"}}
		pdf.CreateMangaPdf(chapterPath, savedFilesByVol, downData.Del)
	}

	mangaID := tools.GetMD5(downData.MangaURL)
	history.SaveHistory(mangaID, saveChapters)

	data.WSChan <- data.WSData{
		Cmd:     "downloadComplete",
		Payload: map[string]interface{}{"text": "Скачивание завершено!"},
	}
	return nil
}

func DownloadChapter(downData data.DownloadOpts, curChapter data.ChaptersList, slug string) ([]string, error) {
	data.WSChan <- data.WSData{
		Cmd: "updateLog",
		Payload: map[string]interface{}{"type": "std", "text": "Скачиваю главу: " + curChapter.Path},
	}

	parts := strings.Split(curChapter.Path, "/")
	volume := strings.TrimLeft(parts[0], "v")
	number := strings.TrimLeft(parts[1], "c")

	apiURL := fmt.Sprintf("%s/%s/chapter?number=%s&volume=%s", apiBase, slug, number, volume)
	body, err := apiRequest(apiURL)
	if err != nil {
		return nil, err
	}

	var resp chapterResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		slog.Error("Ошибка при разборе страниц главы",
			slog.String("chapter", curChapter.Path),
			slog.String("Message", err.Error()),
		)
		return nil, fmt.Errorf("ошибка разбора страниц главы %s: %w", curChapter.Path, err)
	}

	if len(resp.Data.Pages) == 0 {
		slog.Warn("Страницы главы не найдены", slog.String("chapter", curChapter.Path))
		return nil, ErrEmptyPages
	}

	slog.Info("Страницы главы получены",
		slog.String("chapter", curChapter.Path),
		slog.Int("pages", len(resp.Data.Pages)),
	)

	chapterPath := path.Join(config.Cfg.Savepath, downData.SavePath, curChapter.Path)
	if _, err := os.Stat(chapterPath); os.IsNotExist(err) {
		os.MkdirAll(chapterPath, 0755)
	}

	var savedFiles []string

	for i, page := range resp.Data.Pages {
		imgURL := imgBase + page.URL

		client := grab.NewClient()
		client.UserAgent = config.Cfg.UserAgent

		req, err := grab.NewRequest(chapterPath, imgURL)
		if err != nil {
			slog.Error("Ошибка создания запроса страницы",
				slog.String("chapter", curChapter.Path),
				slog.Int("page", i+1),
				slog.String("Message", err.Error()),
			)
			data.WSChan <- data.WSData{
				Cmd: "updateLog",
				Payload: map[string]interface{}{
					"type": "err",
					"text": fmt.Sprintf("-- Ошибка запроса страницы %d: %s", i+1, err.Error()),
				},
			}
			continue
		}
		req.HTTPRequest.Header.Set("Referer", "https://mangalib.me/")

		dlResp := client.Do(req)
		if dlResp.Err() != nil {
			slog.Error("Ошибка скачивания страницы",
				slog.String("chapter", curChapter.Path),
				slog.Int("page", i+1),
				slog.String("url", imgURL),
				slog.String("Message", dlResp.Err().Error()),
			)
			data.WSChan <- data.WSData{
				Cmd: "updateLog",
				Payload: map[string]interface{}{
					"type": "err",
					"text": fmt.Sprintf("-- Ошибка скачивания страницы %d: %s", i+1, dlResp.Err().Error()),
				},
			}
			continue
		}

		savedFiles = append(savedFiles, dlResp.Filename)
		time.Sleep(time.Duration(config.Cfg.Mangalib.TimeoutImage) * time.Millisecond)
	}

	if len(savedFiles) == 0 {
		return nil, fmt.Errorf("не удалось скачать ни одной страницы главы %s", curChapter.Path)
	}

	slog.Info("Глава скачана",
		slog.String("chapter", curChapter.Path),
		slog.Int("saved", len(savedFiles)),
		slog.Int("total", len(resp.Data.Pages)),
	)

	if downData.CBZ == "1" {
		data.WSChan <- data.WSData{Cmd: "updateLog", Payload: map[string]interface{}{"type": "std", "text": "- Создаю CBZ для главы"}}
		tools.CreateCBZ(chapterPath)
	}
	if downData.PDFch == "1" {
		data.WSChan <- data.WSData{Cmd: "updateLog", Payload: map[string]interface{}{"type": "std", "text": "- Создаю PDF для главы"}}
		pdf.CreatePDF(chapterPath, savedFiles)
	}
	if downData.PDFvol != "1" && downData.PDFall != "1" && downData.Del == "1" {
		if err := os.RemoveAll(chapterPath); err != nil {
			slog.Error("Ошибка при удалении файлов", slog.String("Message", err.Error()))
		}
	}

	return savedFiles, nil
}
