[English](localization.md) | [繁體中文](zh-TW/localization.md) | [简体中文](zh-CN/localization.md)

# Language selection

WinToolBox supports English, Traditional Chinese and Simplified Chinese. English is the default/fallback and primary documentation language.

The native first-launch WebView2 download/install window reads the Windows user's display language. Mainland China/Singapore map to Simplified Chinese; Taiwan/Hong Kong/Macao map to Traditional Chinese; other languages map to English. WinToolBox itself runs directly as an EXE.

The application uses the same Windows preference on its first launch. Select **English**, **繁體中文** or **简体中文** in the sidebar to save a manual preference. It takes effect immediately, including Element Plus dialogs and cached module views, and persists across restarts. If local storage is unavailable, selection still works for the current session. Browser development uses the browser's preferred language when the native bridge is absent.

Documentation links select the same topic in another language. Existing shared screenshots show the earlier Simplified Chinese interface. OS command output, time zone/device/user names and GitHub release notes keep their source language.
