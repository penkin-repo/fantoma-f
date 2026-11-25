import { useState, useEffect, useCallback, useRef } from 'react'
import {
    GetFolders,
    AddFolder,
    RemoveFolder,
    RefreshFolder,
    ClearCache,
    GetFamiliesInFolder,
    ActivateFamily,
    DeactivateFamily,
    ActivateFolder,
    DeactivateFolder,
    ActivateFont,
    DeactivateFont,
    ToggleAutostart,
    IsAutostartEnabled,
    GetTheme,
    SetTheme,
    GetRestoreOnStartup,
    SetRestoreOnStartup,
    SetFontCategory,
} from '../../wailsjs/go/main/App'
import { clearLoadedFonts } from '../utils/fontLoader'

export function useFonts() {
    const [folders, setFolders] = useState([])
    const [selectedFolderId, setSelectedFolderId] = useState('all')
    const [families, setFamilies] = useState([])
    const [loading, setLoading] = useState(false)
    const [activeActions, setActiveActions] = useState({})
    const [toast, setToast] = useState(null) // { message, type }

    // Settings & Navigation
    const [view, setView] = useState('list')
    const [isAutostart, setIsAutostart] = useState(false)
    const [restoreOnStartup, setRestoreOnStartupState] = useState(true)
    const [theme, setThemeState] = useState('dark')
    const [previewText, setPreviewText] = useState('')
    const [globalFontSize, setGlobalFontSize] = useState(32)
    const [searchQuery, setSearchQuery] = useState('')
    const [showActiveOnly, setShowActiveOnly] = useState(false)
    const [filters, setFilters] = useState({ category: null, weight: null, hasItalic: false })

    const loadFamiliesRef = useRef(null)
    const loadFoldersRef = useRef(null)
    const loadSeqRef = useRef(0)

    const notify = useCallback((message, type = 'error') => {
        setToast({ message: String(message), type })
    }, [])

    const clearToast = useCallback(() => setToast(null), [])

    const loadFolders = useCallback(async () => {
        try {
            const list = await GetFolders()
            setFolders(list || [])
        } catch (e) {
            notify(e, 'error')
        }
    }, [notify])

    const loadSettings = useCallback(async () => {
        try {
            const [autostart, savedTheme, restore] = await Promise.all([
                IsAutostartEnabled(),
                GetTheme().catch(() => 'dark'),
                GetRestoreOnStartup().catch(() => true),
            ])
            setIsAutostart(autostart)
            const effective = savedTheme === 'light' ? 'light' : 'dark'
            setThemeState(effective)
            document.documentElement.dataset.theme = effective
            setRestoreOnStartupState(Boolean(restore))
        } catch (e) {
            console.error(e)
        }
    }, [])

    useEffect(() => {
        document.documentElement.dataset.theme = 'dark'
        loadFolders()
        loadSettings()
    }, [loadFolders, loadSettings])

    const addNewFolder = async () => {
        try {
            const folder = await AddFolder()
            if (folder) {
                await loadFolders()
                setSelectedFolderId(folder.id)
                setView('list')
            }
        } catch (e) {
            notify(e, 'error')
        }
    }

    const deleteFolder = async (id) => {
        try {
            await RemoveFolder(id)
            await loadFolders()
            if (selectedFolderId === id) {
                setSelectedFolderId('all')
            } else if (selectedFolderId === 'all') {
                loadFamiliesRef.current?.()
            }
        } catch (e) {
            notify(e, 'error')
        }
    }

    const loadFamilies = useCallback(async () => {
        if (view !== 'list') return
        const seq = ++loadSeqRef.current
        setLoading(true)
        try {
            const list = await GetFamiliesInFolder(selectedFolderId)
            // Ignore stale response if user already switched folders
            if (seq !== loadSeqRef.current) return
            setFamilies(list || [])
        } catch (e) {
            if (seq === loadSeqRef.current) {
                notify(e, 'error')
            }
        } finally {
            if (seq === loadSeqRef.current) {
                setLoading(false)
            }
        }
    }, [selectedFolderId, view, notify])

    useEffect(() => {
        loadFamilies()
    }, [loadFamilies])

    useEffect(() => { loadFamiliesRef.current = loadFamilies }, [loadFamilies])
    useEffect(() => { loadFoldersRef.current = loadFolders }, [loadFolders])

    // Subscribe to backend events: fsnotify folder changes & tray DeactivateAll
    useEffect(() => {
        if (!window.runtime?.EventsOn) return
        const unsubFolders = window.runtime.EventsOn('folders:changed', () => {
            loadFamiliesRef.current?.()
            loadFoldersRef.current?.()
        })
        const unsubFonts = window.runtime.EventsOn('fonts:changed', () => {
            loadFamiliesRef.current?.()
            loadFoldersRef.current?.()
        })
        return () => {
            if (typeof unsubFolders === 'function') unsubFolders()
            if (typeof unsubFonts === 'function') unsubFonts()
        }
    }, [])

    const refreshFolder = useCallback(async (id) => {
        try {
            await RefreshFolder(id)
            await loadFamilies()
            await loadFolders()
            notify('Folder refreshed', 'success')
        } catch (e) {
            notify(e, 'error')
        }
    }, [loadFamilies, loadFolders, notify])

    const setAction = (id, val) => setActiveActions(prev => ({ ...prev, [id]: val }))

    const toggleFamily = async (family) => {
        setAction(family.name, true)

        const targetState = !family.isFullyActive
        setFamilies(prev => prev.map(f => {
            if (f.name === family.name) {
                return {
                    ...f,
                    isActive: targetState,
                    isFullyActive: targetState,
                    styles: f.styles.map(s => ({ ...s, isActive: targetState }))
                }
            }
            return f
        }))

        try {
            if (family.isFullyActive) {
                await DeactivateFamily(family.styles)
            } else {
                await ActivateFamily(family.styles)
            }
            await loadFolders()
        } catch (e) {
            notify(e, 'error')
            await loadFamilies()
        } finally {
            setAction(family.name, false)
        }
    }

    const toggleStyle = async (familyName, style) => {
        const actionKey = style.filePath
        setAction(actionKey, true)

        const nextActive = !style.isActive
        setFamilies(prev => prev.map(f => {
            if (f.name !== familyName) return f
            const nextStyles = f.styles.map(s =>
                s.filePath === style.filePath ? { ...s, isActive: nextActive } : s
            )
            return {
                ...f,
                styles: nextStyles,
                isActive: nextStyles.some(s => s.isActive),
                isFullyActive: nextStyles.length > 0 && nextStyles.every(s => s.isActive),
            }
        }))

        try {
            if (style.isActive) {
                await DeactivateFont(style.filePath)
            } else {
                await ActivateFont(style.filePath)
            }
            await loadFolders()
        } catch (e) {
            notify(e, 'error')
            await loadFamilies()
        } finally {
            setAction(actionKey, false)
        }
    }

    const toggleFolder = async (folderId, activate) => {
        setAction(folderId, true)

        // Only optimistically update displayed families if viewing that exact folder
        if (selectedFolderId === folderId) {
            setFamilies(prev => prev.map(f => ({
                ...f,
                isActive: activate,
                isFullyActive: activate,
                styles: f.styles.map(s => ({ ...s, isActive: activate }))
            })))
        }

        try {
            if (activate) {
                await ActivateFolder(folderId)
            } else {
                await DeactivateFolder(folderId)
            }
            await loadFamilies()
            await loadFolders()
        } catch (e) {
            notify(e, 'error')
            await loadFamilies()
        } finally {
            setAction(folderId, false)
        }
    }

    const updateAutostart = async (val) => {
        try {
            await ToggleAutostart(val)
            setIsAutostart(val)
        } catch (e) {
            notify(e, 'error')
        }
    }

    const updateRestoreOnStartup = async (val) => {
        try {
            await SetRestoreOnStartup(val)
            setRestoreOnStartupState(val)
        } catch (e) {
            notify(e, 'error')
        }
    }

    const updateTheme = async (nextTheme) => {
        const effective = nextTheme === 'light' ? 'light' : 'dark'
        setThemeState(effective)
        document.documentElement.dataset.theme = effective
        try {
            await SetTheme(effective)
        } catch (e) {
            notify(e, 'error')
        }
    }

    const setFontCategory = useCallback(async (familyName, category) => {
        setFamilies(prev => prev.map(f =>
            f.name === familyName
                ? { ...f, category: category || f.category, isManual: category !== '' }
                : f
        ))
        try {
            await SetFontCategory(familyName, category)
            await loadFamilies()
        } catch (e) {
            notify(e, 'error')
        }
    }, [loadFamilies, notify])

    const clearCache = useCallback(async () => {
        try {
            clearLoadedFonts()
            await ClearCache()
            await loadFamilies()
            notify('Font preview & metadata cache cleared', 'success')
        } catch (e) {
            notify(e, 'error')
        }
    }, [loadFamilies, notify])

    const WEIGHT_RANGE = 100

    const filteredFamilies = families
        .filter(f => !searchQuery || f.name.toLowerCase().includes(searchQuery.toLowerCase()))
        .filter(f => !showActiveOnly || f.isActive)
        .filter(f => !filters.category || f.category === filters.category)
        .filter(f => !filters.weight || Math.abs((f.weightClass || 400) - filters.weight) < WEIGHT_RANGE)
        .filter(f => !filters.hasItalic || f.hasItalic)

    return {
        folders,
        selectedFolderId,
        setSelectedFolderId,
        addNewFolder,
        deleteFolder,
        refreshFolder,
        families: filteredFamilies,
        totalFamiliesCount: families.length,
        loading,
        activeActions,
        toast,
        clearToast,
        searchQuery,
        setSearchQuery,
        showActiveOnly,
        setShowActiveOnly,
        filters,
        setFilters,
        setFontCategory,
        previewText,
        setPreviewText,
        globalFontSize,
        setGlobalFontSize,
        toggleFamily,
        toggleStyle,
        toggleFolder,
        view,
        setView,
        isAutostart,
        updateAutostart,
        restoreOnStartup,
        updateRestoreOnStartup,
        theme,
        updateTheme,
        clearCache,
    }
}
