
## Developer

### `.vscode/settings.json`

```json
{
    "files.exclude": {
        "vendor": true,
        "tmp": false,
        "bin": true,
    },
    "go.formatTool": "gofmt",
    "go.formatFlags": [
        "-s"
    ],
    "go.testFlags": [
        "-v",
        "-count", "1"
    ]
}
```