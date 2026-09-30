# FLiNG Trainer Browser

FLiNG Trainer Browser is a Wails 3 desktop application for searching, viewing,
and downloading trainers from flingtrainer.com.

## Development

Run from the project root:

```powershell
wails3 dev -config .\build\config.yml -port 9245
```

Regenerate bindings after changing exported Go services:

```powershell
wails3 generate bindings -ts -i
```

## Production build

Build the Windows application with:

```powershell
wails3 build -config .\build\config.yml
```

The executable is written to:

```text
bin\fling-trainer-browser.exe
```

The frontend production bundle is embedded into the executable. The Vite
configuration splits React, Ant Design, and Wails runtime dependencies into
separate chunks.

Cross-platform builds can be requested with the target-specific Wails build
options. Native signing and packaging requirements still apply:

- Windows: unsigned binaries may trigger Microsoft Defender reputation warnings.
  Code-sign the executable and distribute a checksum with releases.
- macOS: an outside-App-Store build needs Developer ID signing and notarization
  before Gatekeeper will open it without an override.
- Linux: package the executable using the distribution format appropriate for
  the target system.

## Project structure

- `backend/models`: Shared trainer data models.
- `backend/scraper`: Search and detail-page scraping.
- `backend/services`: Wails services and download handling.
- `frontend/src`: React UI, Zustand state, and Wails event handling.
- `frontend/bindings`: Generated TypeScript bindings.
- `main.go`: Application entry point and service registration.

Downloaded archives are stored in:

```text
~/Downloads/FLiNG_Trainers/
```
