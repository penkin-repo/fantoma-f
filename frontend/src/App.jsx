import { useRef, useEffect, useState, useCallback } from 'react'
import { useFonts } from './hooks/useFonts'
import Sidebar from './components/Sidebar'
import TopBar from './components/TopBar'
import FilterBar from './components/FilterBar'
import FontRow from './components/FontRow'
import CategoryPicker from './components/CategoryPicker'
import SettingsView from './components/SettingsView'
import Toast from './components/Toast'
import { useVirtualizer } from '@tanstack/react-virtual'
import { FolderOpen } from 'lucide-react'

export default function App() {
    const {
        folders,
        selectedFolderId,
        setSelectedFolderId,
        addNewFolder,
        deleteFolder,
        refreshFolder,
        families,
        loading,
        activeActions,
        toast,
        clearToast,
        searchQuery,
        setSearchQuery,
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
        showActiveOnly,
        setShowActiveOnly,
        filters,
        setFilters,
        setFontCategory,
    } = useFonts()

    const [categoryPicker, setCategoryPicker] = useState(null) // { family, x, y }
    const [expandedFamilies, setExpandedFamilies] = useState({})

    const handleCategoryClick = useCallback((family, e) => {
        const rect = e.currentTarget.getBoundingClientRect()
        const x = Math.min(rect.left, window.innerWidth - 230)
        const y = Math.min(rect.bottom + 6, window.innerHeight - 290)
        setCategoryPicker({ family, x, y })
    }, [])

    const handleCategorySelect = useCallback(async (category) => {
        if (categoryPicker) {
            await setFontCategory(categoryPicker.family.name, category)
        }
        setCategoryPicker(null)
    }, [categoryPicker, setFontCategory])

    const handleToggleExpand = useCallback((familyName) => {
        setExpandedFamilies(prev => ({ ...prev, [familyName]: !prev[familyName] }))
    }, [])

    const parentRef = useRef()

    const [debouncedFontSize, setDebouncedFontSize] = useState(globalFontSize)
    useEffect(() => {
        const t = setTimeout(() => setDebouncedFontSize(globalFontSize), 250)
        return () => clearTimeout(t)
    }, [globalFontSize])

    const virtualizer = useVirtualizer({
        count: families.length,
        getScrollElement: () => parentRef.current,
        estimateSize: () => Math.max(76, debouncedFontSize + 48),
        measureElement: (el) => el?.getBoundingClientRect().height ?? Math.max(76, debouncedFontSize + 48),
        overscan: 8,
    })

    useEffect(() => {
        virtualizer.measure()
    }, [debouncedFontSize, expandedFamilies, virtualizer])

    // Trim OS working set when window is hidden in system tray
    useEffect(() => {
        const handleVisibilityChange = () => {
            if (document.hidden && window.go?.main?.App?.TrimMemory) {
                window.go.main.App.TrimMemory()
            }
        }
        document.addEventListener('visibilitychange', handleVisibilityChange)
        return () => document.removeEventListener('visibilitychange', handleVisibilityChange)
    }, [])

    return (
        <div className="app-container">
            <Sidebar
                folders={folders}
                selectedId={selectedFolderId}
                onSelect={setSelectedFolderId}
                onAdd={addNewFolder}
                onRemove={deleteFolder}
                onRefresh={refreshFolder}
                onToggleFolder={toggleFolder}
                activeActions={activeActions}
                searchQuery={searchQuery}
                setSearchQuery={setSearchQuery}
                view={view}
                setView={setView}
                theme={theme}
                onToggleTheme={() => updateTheme(theme === 'dark' ? 'light' : 'dark')}
            />

            <main className="main-content">
                {view === 'settings' ? (
                    <SettingsView
                        isAutostart={isAutostart}
                        onToggleAutostart={updateAutostart}
                        restoreOnStartup={restoreOnStartup}
                        onToggleRestoreOnStartup={updateRestoreOnStartup}
                        theme={theme}
                        onChangeTheme={updateTheme}
                        onClearCache={clearCache}
                    />
                ) : (
                    <>
                        <TopBar
                            previewText={previewText}
                            setPreviewText={setPreviewText}
                            fontSize={globalFontSize}
                            setFontSize={setGlobalFontSize}
                            fontCount={families.length}
                            showActiveOnly={showActiveOnly}
                            setShowActiveOnly={setShowActiveOnly}
                        />
                        <FilterBar filters={filters} setFilters={setFilters} />

                        <div className="font-list" ref={parentRef}>
                            {loading ? (
                                <div className="list-message">
                                    <div className="spinner-large" />
                                    <p>Scanning fonts...</p>
                                </div>
                            ) : families.length === 0 ? (
                                <div className="list-message">
                                    <FolderOpen size={36} strokeWidth={1.5} className="empty-icon" />
                                    <p>
                                        {searchQuery || showActiveOnly || filters.category || filters.weight || filters.hasItalic
                                            ? 'No fonts match your current filters.'
                                            : 'No fonts found in this folder.'}
                                    </p>
                                </div>
                            ) : (
                                <div
                                    className="virtual-container"
                                    style={{
                                        height: `${virtualizer.getTotalSize()}px`,
                                        width: '100%',
                                        position: 'relative',
                                    }}
                                >
                                    {virtualizer.getVirtualItems().map((virtualItem) => {
                                        const family = families[virtualItem.index]
                                        return (
                                            <div
                                                key={virtualItem.key}
                                                data-index={virtualItem.index}
                                                ref={virtualizer.measureElement}
                                                style={{
                                                    position: 'absolute',
                                                    top: 0,
                                                    left: 0,
                                                    width: '100%',
                                                    transform: `translateY(${virtualItem.start}px)`,
                                                }}
                                            >
                                                <FontRow
                                                    family={family}
                                                    onToggle={toggleFamily}
                                                    onToggleStyle={toggleStyle}
                                                    onCategoryClick={handleCategoryClick}
                                                    previewText={previewText}
                                                    fontSize={debouncedFontSize}
                                                    isToggling={activeActions[family.name]}
                                                    activeActions={activeActions}
                                                    expanded={Boolean(expandedFamilies[family.name])}
                                                    onToggleExpand={() => handleToggleExpand(family.name)}
                                                />
                                            </div>
                                        )
                                    })}
                                </div>
                            )}
                        </div>
                    </>
                )}
            </main>

            {toast && <Toast message={toast.message} type={toast.type} onClose={clearToast} />}

            {categoryPicker && (
                <CategoryPicker
                    family={categoryPicker.family}
                    position={{ x: categoryPicker.x, y: categoryPicker.y }}
                    onSelect={handleCategorySelect}
                    onClose={() => setCategoryPicker(null)}
                />
            )}

            <style>{`
                .app-container {
                    display: flex;
                    height: 100vh;
                    width: 100vw;
                    background-color: var(--bg-app);
                    color: var(--text-main);
                }

                .main-content {
                    flex: 1;
                    display: flex;
                    flex-direction: column;
                    overflow: hidden;
                    background-color: var(--bg-app);
                    min-width: 0;
                }

                .font-list {
                    flex: 1;
                    overflow-y: auto;
                    background-color: var(--bg-app);
                }

                .list-message {
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    justify-content: center;
                    height: 320px;
                    gap: 14px;
                    color: var(--text-muted);
                    font-size: 14px;
                }

                .empty-icon {
                    opacity: 0.45;
                    color: var(--accent);
                }

                .spinner-large {
                    width: 30px;
                    height: 30px;
                    border: 2.5px solid var(--border-strong);
                    border-top-color: var(--accent);
                    border-radius: 50%;
                    animation: spin 0.75s linear infinite;
                }
            `}</style>
        </div>
    )
}
