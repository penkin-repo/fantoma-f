// Lazy, deduplicated FontFace loader with collision-free Unicode/Cyrillic IDs.

const loadedFaces = new Map() // id -> FontFace | 'loading'

// FNV-1a 32-bit hash over UTF-16 code units — handles Cyrillic & any Unicode safely.
function hashString(str) {
    let h = 0x811c9dc5
    for (let i = 0; i < str.length; i++) {
        h ^= str.charCodeAt(i)
        h = Math.imul(h, 0x01000193)
    }
    return (h >>> 0).toString(36)
}

export function fontFaceId(key) {
    return `ff-f-${hashString(String(key || ''))}`
}

export function ensureFontLoaded(id, filePath) {
    if (!filePath || loadedFaces.has(id)) return
    loadedFaces.set(id, 'loading')

    const fontUrl = `http://wails.localhost/font?path=${encodeURIComponent(filePath)}`
    const ff = new FontFace(id, `url('${fontUrl}')`)
    ff.load()
        .then((loaded) => {
            if (loadedFaces.get(id) === 'loading') {
                document.fonts.add(loaded)
                loadedFaces.set(id, loaded)
            }
        })
        .catch(() => {
            if (loadedFaces.get(id) === 'loading') {
                loadedFaces.delete(id)
            }
        })
}

export function clearLoadedFonts() {
    for (const face of loadedFaces.values()) {
        if (face && face !== 'loading') {
            try {
                document.fonts.delete(face)
            } catch (_) {
                // ignore
            }
        }
    }
    loadedFaces.clear()
}
