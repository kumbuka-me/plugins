// Package localize contains shared first-party plugin interface messages.
package localize

import sdk "github.com/kumbuka-me/sdk"

var translations = sdk.Translations{
	"en": {
		"time.just_now": "just now", "time.minutes_ago": "%dm ago", "time.hours_ago": "%dh ago", "time.days_ago": "%dd ago",
		"common.views": "%d views", "common.pages": "Pages", "common.untitled": "Untitled",
		"continue.title": "Continue working", "continue.drafts": "Drafts", "continue.no_drafts": "No private drafts.", "continue.edits": "Recent edits", "continue.no_edits": "Pages you edit will appear here.", "continue.private_draft": "Private draft", "continue.stale": "Page changed since draft started",
		"favorites.title": "Favorites", "favorites.empty": "Star useful pages to keep them close.", "favorites.pinned": "Pinned",
		"popular.title": "Popular pages", "popular.empty": "Popular pages appear after they are viewed.",
		"recent.title": "Recent changes", "recent.view_all": "View all", "recent.ready": "Your Kumbuka knowledge base is ready.", "recent.empty": "Create the first page and start linking knowledge together.",
		"viewed.title": "Recently viewed", "viewed.empty": "Pages you open will appear here.",
		"related.title": "Related pages", "related.empty": "No related pages yet.",
		"revision.title": "Revision history", "revision.empty": "No revision history available.", "revision.lines": "lines", "revision.created": "created page", "revision.metadata": "metadata-only save",
		"wiki.referenced_by": "Referenced by", "wiki.no_backlinks": "No pages link here yet.", "wiki.links_from": "Links from this page", "wiki.no_links": "No wiki links on this page.", "wiki.missing": "Missing page",
		"include.from":     "Included from",
		"subpages.title":   "Pages in this section",
		"external.invalid": "Invalid external file. Use source and path, optional lines, notes, and supported presentation overrides.", "external.unavailable_admin": "External file unavailable. Ask an administrator to check the configured source and provider access.", "external.unavailable_range": "External file unavailable. Check the requested line range.", "external.annotation_range": "An annotation refers to a line outside the displayed file range.", "external.annotation": "Annotation %d", "external.line": "Line", "external.lines": "Lines",
		"status.error": "Status error: %s", "status.title": "Status: %s", "status.controls": "Status controls", "status.controls_help": "Change page statuses without editing the Markdown.", "status.current": "Current: %s",
		"tasks.error": "Task error: %s", "tasks.due": "Due %s", "tasks.title": "Tasks", "tasks.help": "Track tasks on this page without editing its Markdown.", "tasks.move": "Move to %s: %s",
	},
	"de": {
		"time.just_now": "gerade eben", "time.minutes_ago": "vor %d Min.", "time.hours_ago": "vor %d Std.", "time.days_ago": "vor %d Tagen",
		"common.views": "%d Aufrufe", "common.pages": "Seiten", "common.untitled": "Ohne Titel",
		"continue.title": "Weiterarbeiten", "continue.drafts": "Entwürfe", "continue.no_drafts": "Keine privaten Entwürfe.", "continue.edits": "Letzte Bearbeitungen", "continue.no_edits": "Bearbeitete Seiten erscheinen hier.", "continue.private_draft": "Privater Entwurf", "continue.stale": "Seite wurde seit Beginn des Entwurfs geändert",
		"favorites.title": "Favoriten", "favorites.empty": "Markiere nützliche Seiten, um sie griffbereit zu halten.", "favorites.pinned": "Angeheftet",
		"popular.title": "Beliebte Seiten", "popular.empty": "Beliebte Seiten erscheinen, nachdem sie aufgerufen wurden.",
		"recent.title": "Letzte Änderungen", "recent.view_all": "Alle anzeigen", "recent.ready": "Deine Kumbuka-Wissensdatenbank ist bereit.", "recent.empty": "Erstelle die erste Seite und verknüpfe dein Wissen.",
		"viewed.title": "Zuletzt angesehen", "viewed.empty": "Geöffnete Seiten erscheinen hier.",
		"related.title": "Verwandte Seiten", "related.empty": "Noch keine verwandten Seiten.",
		"revision.title": "Versionsverlauf", "revision.empty": "Kein Versionsverlauf verfügbar.", "revision.lines": "Zeilen", "revision.created": "Seite erstellt", "revision.metadata": "nur Metadaten gespeichert",
		"wiki.referenced_by": "Verlinkt von", "wiki.no_backlinks": "Noch keine Seiten verlinken hierher.", "wiki.links_from": "Links von dieser Seite", "wiki.no_links": "Keine Wiki-Links auf dieser Seite.", "wiki.missing": "Fehlende Seite",
		"include.from":     "Eingebunden von",
		"subpages.title":   "Seiten in diesem Abschnitt",
		"external.invalid": "Ungültige externe Datei. Verwende Quelle und Pfad sowie optional Zeilen, Hinweise und unterstützte Darstellungsoptionen.", "external.unavailable_admin": "Externe Datei nicht verfügbar. Bitte einen Administrator, die konfigurierte Quelle und den Anbieterzugriff zu prüfen.", "external.unavailable_range": "Externe Datei nicht verfügbar. Prüfe den angeforderten Zeilenbereich.", "external.annotation_range": "Ein Hinweis verweist auf eine Zeile ausserhalb des angezeigten Dateibereichs.", "external.annotation": "Hinweis %d", "external.line": "Zeile", "external.lines": "Zeilen",
		"status.error": "Statusfehler: %s", "status.title": "Status: %s", "status.controls": "Statussteuerung", "status.controls_help": "Ändere Seitenstatus, ohne das Markdown zu bearbeiten.", "status.current": "Aktuell: %s",
		"tasks.error": "Aufgabenfehler: %s", "tasks.due": "Fällig %s", "tasks.title": "Aufgaben", "tasks.help": "Verwalte Aufgaben auf dieser Seite, ohne das Markdown zu bearbeiten.", "tasks.move": "Auf %s setzen: %s",
	},
}

// For returns the shared plugin localizer for one host-selected locale.
func For(locale string) sdk.Localizer { return sdk.NewLocalizer(locale, translations) }
