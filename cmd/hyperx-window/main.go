// hyperx-studio-window показывает интерфейс отдельным окном на GTK и WebKit.
//
// Раньше окно рисовал Chrome в режиме приложения. Это работало, но окно
// оставалось чужим: имя ему давал браузер (chrome-127.0.0.1__-Default),
// иконку он брал свою, сверху висела браузерная полоса с заголовком, а
// вместе с окном приходил весь Chromium со своими приветствиями.
//
// Здесь окно наше. Имя и иконку берёт оболочка из /usr/share/applications,
// заголовок в полосе пустой — остаются только кнопки окна, — а страницу
// показывает WebKit, который в GNOME уже установлен.
//
// Это отдельная программа, а не часть hyperx-studio, и на то две причины.
// Подсветкой заведует служба, которая работает и без окна: тянуть в неё
// сотню мегабайт GTK и WebKit только ради того, чтобы их не открывать, —
// расточительно. И падение WebKit при этом гасит окно, а не подсветку.
package main

/*
#cgo pkg-config: gtk+-3.0 webkit2gtk-4.1
#include <gtk/gtk.h>
#include <glib-unix.h>
#include <webkit2/webkit2.h>
#include <stdlib.h>

// Полосу заголовка красим под интерфейс: со светлой системной темой она
// выглядела бы приклеенной сверху чужой деталью.
static const char *HXS_CSS =
    "window { background-color: #0B0D12; }"
    "headerbar {"
    "  background-image: none;"
    "  background-color: #12151D;"
    "  border-bottom: 1px solid #232838;"
    "  box-shadow: none;"
    "  min-height: 30px;"
    "  padding: 0 4px;"
    "}"
    "headerbar button.titlebutton {"
    "  color: #99A0B4; min-width: 26px; min-height: 26px;"
    "  background: none; border: 0; box-shadow: none;"
    "}"
    "headerbar button.titlebutton:hover { color: #E7EAF2; }";

static GtkWidget *hxs_window = NULL;

static void hxs_on_destroy(GtkWidget *w, gpointer data) {
    (void)w; (void)data;
    hxs_window = NULL;
    gtk_main_quit();
}

// Повторный запуск программы не открывает второе окно, а просит показать
// это: служба шлёт SIGUSR1.
static gboolean hxs_on_present(gpointer data) {
    (void)data;
    if (hxs_window != NULL) gtk_window_present(GTK_WINDOW(hxs_window));
    return TRUE;
}

static gboolean hxs_on_term(gpointer data) {
    (void)data;
    if (hxs_window != NULL) gtk_widget_destroy(hxs_window);
    gtk_main_quit();
    return FALSE;
}

// hxs_run открывает окно и возвращается, когда его закрыли. Ноль означает,
// что окна нет — например, программу запустили без графической сессии.
static int hxs_run(const char *url, const char *title, const char *appid,
                   int width, int height) {
    // Под Wayland оболочка ищет запись .desktop по имени программы —
    // отсюда берутся и подпись в панели задач, и иконка.
    g_set_prgname(appid);
    if (!gtk_init_check(NULL, NULL)) return 0;
    gdk_set_program_class(title);

    g_object_set(gtk_settings_get_default(),
                 "gtk-application-prefer-dark-theme", TRUE, NULL);

    GtkCssProvider *css = gtk_css_provider_new();
    gtk_css_provider_load_from_data(css, HXS_CSS, -1, NULL);
    gtk_style_context_add_provider_for_screen(
        gdk_screen_get_default(), GTK_STYLE_PROVIDER(css),
        GTK_STYLE_PROVIDER_PRIORITY_APPLICATION);
    g_object_unref(css);

    GtkWidget *win = gtk_window_new(GTK_WINDOW_TOPLEVEL);
    hxs_window = win;
    gtk_window_set_default_size(GTK_WINDOW(win), width, height);
    gtk_window_set_title(GTK_WINDOW(win), title);
    gtk_window_set_icon_name(GTK_WINDOW(win), appid);

    // Пустая подпись вместо заголовка: имя программы и так написано в
    // панели задач, а над самой программой оно только занимает место.
    GtkWidget *bar = gtk_header_bar_new();
    gtk_header_bar_set_show_close_button(GTK_HEADER_BAR(bar), TRUE);
    gtk_header_bar_set_custom_title(GTK_HEADER_BAR(bar), gtk_label_new(""));
    gtk_window_set_titlebar(GTK_WINDOW(win), bar);

    GtkWidget *view = webkit_web_view_new();
    GdkRGBA bg = {0.043, 0.051, 0.071, 1.0};
    webkit_web_view_set_background_color(WEBKIT_WEB_VIEW(view), &bg);

    WebKitSettings *set = webkit_web_view_get_settings(WEBKIT_WEB_VIEW(view));
    webkit_settings_set_enable_developer_extras(set, FALSE);
    webkit_settings_set_enable_page_cache(set, FALSE);
    webkit_settings_set_javascript_can_open_windows_automatically(set, FALSE);

    webkit_web_view_load_uri(WEBKIT_WEB_VIEW(view), url);
    gtk_container_add(GTK_CONTAINER(win), view);

    g_signal_connect(win, "destroy", G_CALLBACK(hxs_on_destroy), NULL);
    g_unix_signal_add(SIGUSR1, hxs_on_present, NULL);
    g_unix_signal_add(SIGTERM, hxs_on_term, NULL);
    g_unix_signal_add(SIGINT, hxs_on_term, NULL);

    gtk_widget_show_all(win);
    gtk_widget_grab_focus(view);

    gtk_main();

    if (hxs_window != NULL) {
        gtk_widget_destroy(hxs_window);
        hxs_window = NULL;
    }
    return 1;
}
*/
import "C"

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"unsafe"
)

func main() {
	title := flag.String("title", "HyperX Studio", "заголовок окна")
	appID := flag.String("app-id", "hyperx-studio", "имя программы для оболочки")
	width := flag.Int("width", 1180, "ширина окна")
	height := flag.Int("height", 760, "высота окна")
	flag.Parse()

	url := flag.Arg(0)
	if url == "" {
		fmt.Fprintln(os.Stderr, "нужен адрес интерфейса")
		os.Exit(2)
	}

	// GTK живёт только на главном потоке операционной системы, а
	// планировщик Go переносит горутины куда захочет.
	runtime.LockOSThread()

	cURL := C.CString(url)
	cTitle := C.CString(*title)
	cAppID := C.CString(*appID)
	defer func() {
		C.free(unsafe.Pointer(cURL))
		C.free(unsafe.Pointer(cTitle))
		C.free(unsafe.Pointer(cAppID))
	}()

	if C.hxs_run(cURL, cTitle, cAppID, C.int(*width), C.int(*height)) == 0 {
		fmt.Fprintln(os.Stderr, "не удалось открыть окно: нет графической сессии")
		os.Exit(1)
	}
}
