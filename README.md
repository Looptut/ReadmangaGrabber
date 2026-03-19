Оригинальный автор утилиты: [![Build status](https://api.travis-ci.com/lirix360/ReadmangaGrabber.svg?branch=master)](https://travis-ci.com/github/lirix360/ReadmangaGrabber) [![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg?style=flat)](https://goreportcard.com/report/github.com/lirix360/readmangagrabber)

Утилита для скачивания манги с сайтов ReadManga, MintManga и SelfManga.

## Изменения

Ввиду изменчивости html, зеркал, были произведены некоторые корректировки к способу использования утилиты

## Возможности

* Скачивание целой манги / указанного списка глав из манги
* Создание PDF файлов для скачанных глав
* Создание CBZ файлов для скачанных глав

**Возможности скачивания платной манги нет и не будет!**

## Использование ТОЛЬКО через сборку (потому что у автора изменений не было желания собирать под разные ОС исполняемые файлы)

* Установите [последнюю версию](https://go.dev/dl) языка Go
* Скачайте исходный код в удобное место с помощью git или в виде zip-файла
* Запустите файл сборки соответствующий вашей ОС (build_win.bat, build_linux.sh или build_osx.sh)
* Скомпилированная версия утилиты появится в папке builds/ваша_ОС
* Положите в папку файл src_list.json из корня проекта(если он там по какой-то причине не лежит)

## Если прошло сколько-то времени и появились новые зеркала, главы не видны или куки не подходит
* проверить наличие домена в src_list.json и добавить
* если это не помогает, попробовать в корне проекта еще и lib_urls.json обновить (всё самостоятельно)
* пересобрать
 
**Если возникли проблемы при скачивании глав манги, терминальчик вылетает, процессы крашатся, ухххх**

Нейронка вам в помощь. Скорее всего поменялся html формат страницы, который не получается распарсить, а логами код плохо обложен. Надо запускать через консоль так, чтобы все логи были видны 
(Win+R -> cmd)
```
cd C:\путь\до\папки\с\ReadmangaGrabber
grabber_win_x64.exe
```

Обзор проекта в виде единого txt можно с помощью summarizer.go 
```
go run summarizer.go
```

Потом спрашивать любую доступную нейронку, что там вообще происходит и править на свой страх и риск.

## Сохранение cookies для ReadManga/MintManga

1. Войдите со своими логином/паролем на нужный сайт
2. Для сохранения cookies в файл используйте расширение браузера (например, для Chrome и аналогов: [Get cookies.txt LOCALLY](https://chrome.google.com/webstore/detail/get-cookiestxt-locally/cclelndahbckbenkjhflpdbgdldlbecc), [Cookie-Editor](https://chromewebstore.google.com/detail/cookie-editor/hlkenndednhfkekhgcdicdfddnkalmdm) для FireFox: [cookies.txt](https://addons.mozilla.org/ru/firefox/addon/cookies-txt/))
3. Сохраненный файл переименуйте соответственно домену нужного сайта с расширением .txt, например, readmanga.live.txt или mangalib.me.txt, и положите в папку с программой

В случае если у вас есть сохраненный файл с cookies, но манга не скачивается вероятно истек срок действия cookies, сохраните их еще раз.

![Интерфейс](https://raw.githubusercontent.com/lirix360/ReadmangaGrabber/gh-pages/screenshot-v2.png)
