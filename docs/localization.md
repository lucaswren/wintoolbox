[English](localization.md) | [繁體中文](zh-Hant/localization.md) | [简体中文](zh-Hans/localization.md)

# Language selection

WinToolBox supports English, Traditional Chinese and Simplified Chinese. English is the default/fallback and primary documentation language.

The native first-launch WebView2 download/install window follows the Windows display language. Chinese is identified by writing system: `zh-Hans` for Simplified Chinese and `zh-Hant` for Traditional Chinese. English uses `en`. These identifiers do not include a region. OS language tags are mapped to the corresponding writing system; unsupported languages fall back to English. WinToolBox runs directly as an EXE.

The application uses the same Windows preference on its first launch. Select **English**, **繁體中文** or **简体中文** in the sidebar to save a manual preference. It takes effect immediately, including Element Plus dialogs and cached module views, and persists across restarts. If local storage is unavailable, selection still works for the current session. Browser development uses the browser's preferred language when the native bridge is absent.

Documentation links select the same topic in another language. Existing shared screenshots show the earlier Simplified Chinese interface. OS command output, time zone/device/user names and GitHub release notes keep their source language.
