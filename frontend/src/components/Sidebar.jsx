import {
    Search,
    FolderPlus,
    Layers,
    Folder,
    RefreshCw,
    X,
    Settings,
    Sun,
    Moon,
    Type,
} from 'lucide-react'
import styles from './Sidebar.module.css'

export default function Sidebar({
    folders,
    selectedId,
    onSelect,
    onAdd,
    onRemove,
    onRefresh,
    onToggleFolder,
    activeActions,
    searchQuery,
    setSearchQuery,
    view,
    setView,
    theme,
    onToggleTheme,
}) {
    const isListView = view === 'list'

    const handleSelect = (id) => {
        onSelect(id)
        setView('list')
    }

    return (
        <aside className={styles.sidebar}>
            <div className={styles.header}>
                <div className={styles.brand} onClick={() => handleSelect('all')}>
                    <span className={styles.brandIcon}>
                        <Type size={16} strokeWidth={2.4} />
                    </span>
                    <h1 className={styles.title}>Fantoma F</h1>
                </div>

                <div className={styles.searchBox}>
                    <Search size={14} className={styles.searchIcon} />
                    <input
                        type="text"
                        placeholder="Search fonts..."
                        className={styles.searchInput}
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                    />
                    {searchQuery && (
                        <button
                            className={styles.clearSearch}
                            onClick={() => setSearchQuery('')}
                            title="Clear search"
                        >
                            <X size={13} />
                        </button>
                    )}
                </div>

                <button className={styles.addButton} onClick={onAdd} title="Add Font Folder">
                    <FolderPlus size={15} strokeWidth={2.1} />
                    <span>Add Folder</span>
                </button>
            </div>

            <nav className={styles.folderList}>
                <div
                    className={`${styles.folderItem} ${styles.globalItem} ${isListView && selectedId === 'all' ? styles.active : ''}`}
                    onClick={() => handleSelect('all')}
                >
                    <div className={styles.folderHeaderSingle}>
                        <Layers size={15} className={styles.folderIcon} />
                        <span className={styles.folderName}>All Fonts</span>
                    </div>
                </div>

                <div className={styles.separator}>Folders</div>

                {folders.map(folder => (
                    <div
                        key={folder.id}
                        className={`${styles.folderItem} ${isListView && selectedId === folder.id ? styles.active : ''}`}
                        onClick={() => handleSelect(folder.id)}
                    >
                        <div className={styles.folderHeader}>
                            <span
                                className={`${styles.statusDot} ${styles[folder.status]}`}
                                title={`Status: ${folder.status}`}
                            />
                            <Folder size={14} className={styles.folderIcon} />
                            <span className={styles.folderName} title={folder.path}>{folder.name}</span>
                            <div className={styles.headerIcons}>
                                <button
                                    className={styles.iconBtn}
                                    onClick={(e) => { e.stopPropagation(); onRefresh(folder.id); }}
                                    title="Refresh folder"
                                >
                                    <RefreshCw size={13} />
                                </button>
                                <button
                                    className={`${styles.iconBtn} ${styles.removeBtn}`}
                                    onClick={(e) => { e.stopPropagation(); onRemove(folder.id); }}
                                    title="Remove folder (also deactivates its fonts)"
                                >
                                    <X size={13} />
                                </button>
                            </div>
                        </div>

                        <div className={styles.folderActions}>
                            <button
                                className={`${styles.bulkButton} ${styles.activate}`}
                                onClick={(e) => { e.stopPropagation(); onToggleFolder(folder.id, true); }}
                                disabled={activeActions[folder.id]}
                            >
                                {activeActions[folder.id] ? <div className="spinner" /> : 'ON'}
                            </button>
                            <button
                                className={`${styles.bulkButton} ${styles.deactivate}`}
                                onClick={(e) => { e.stopPropagation(); onToggleFolder(folder.id, false); }}
                                disabled={activeActions[folder.id]}
                            >
                                {activeActions[folder.id] ? <div className="spinner" /> : 'OFF'}
                            </button>
                        </div>
                    </div>
                ))}

                {folders.length === 0 && (
                    <div className={styles.empty}>
                        No folders added yet.
                    </div>
                )}
            </nav>

            <div className={styles.footer}>
                <button
                    className={`${styles.settingsButton} ${view === 'settings' ? styles.settingsActive : ''}`}
                    onClick={() => setView('settings')}
                    title="Settings"
                >
                    <Settings size={16} />
                    <span>Settings</span>
                </button>
                <button
                    className={styles.themeToggle}
                    onClick={onToggleTheme}
                    title={theme === 'dark' ? 'Switch to Light theme' : 'Switch to Dark theme'}
                >
                    {theme === 'dark' ? <Sun size={16} /> : <Moon size={16} />}
                </button>
            </div>
        </aside>
    )
}
