import { useEffect, useState } from 'react'
import { Moon, Sun, Trash2, Power, RotateCcw, Info } from 'lucide-react'
import { GetAppVersion } from '../../wailsjs/go/main/App'
import styles from './SettingsView.module.css'

export default function SettingsView({
    isAutostart,
    onToggleAutostart,
    restoreOnStartup,
    onToggleRestoreOnStartup,
    theme,
    onChangeTheme,
    onClearCache,
}) {
    const [version, setVersion] = useState('')

    useEffect(() => {
        GetAppVersion().then(setVersion).catch(() => {})
    }, [])

    return (
        <div className={styles.container}>
            <header className={styles.header}>
                <h2 className={styles.title}>Application Settings</h2>
                <p className={styles.subtitle}>Configure Fantoma F appearance, startup behavior, and cache.</p>
            </header>

            <section className={styles.section}>
                <h3 className={styles.sectionTitle}>Appearance</h3>
                <div className={styles.themeGrid}>
                    <button
                        className={`${styles.themeCard} ${theme === 'dark' ? styles.themeCardActive : ''}`}
                        onClick={() => onChangeTheme('dark')}
                    >
                        <span className={styles.themeIconWrap}>
                            <Moon size={18} />
                        </span>
                        <div className={styles.themeText}>
                            <strong>Obsidian Dark</strong>
                            <span>Studio dark interface for Illustrator & design work</span>
                        </div>
                    </button>

                    <button
                        className={`${styles.themeCard} ${theme === 'light' ? styles.themeCardActive : ''}`}
                        onClick={() => onChangeTheme('light')}
                    >
                        <span className={styles.themeIconWrap}>
                            <Sun size={18} />
                        </span>
                        <div className={styles.themeText}>
                            <strong>Warm Paper</strong>
                            <span>Soft creamy light interface</span>
                        </div>
                    </button>
                </div>
            </section>

            <section className={styles.section}>
                <h3 className={styles.sectionTitle}>Startup & Lifecycle</h3>
                <div className={styles.cardGroup}>
                    <div className={styles.settingItem}>
                        <div className={styles.settingIcon}>
                            <Power size={17} />
                        </div>
                        <div className={styles.settingInfo}>
                            <div className={styles.settingLabel}>Launch on Startup</div>
                            <div className={styles.settingDescription}>
                                Automatically start Fantoma F minimized in the system tray when Windows boots.
                            </div>
                        </div>
                        <div className={styles.settingAction}>
                            <label className={styles.switch}>
                                <input
                                    type="checkbox"
                                    checked={isAutostart}
                                    onChange={(e) => onToggleAutostart(e.target.checked)}
                                />
                                <span className={styles.slider}></span>
                            </label>
                        </div>
                    </div>

                    <div className={styles.settingItem}>
                        <div className={styles.settingIcon}>
                            <RotateCcw size={17} />
                        </div>
                        <div className={styles.settingInfo}>
                            <div className={styles.settingLabel}>Restore Active Fonts on Startup</div>
                            <div className={styles.settingDescription}>
                                Cleanly unload fonts on shutdown/reboot and automatically re-activate your active set when Fantoma F starts again.
                            </div>
                        </div>
                        <div className={styles.settingAction}>
                            <label className={styles.switch}>
                                <input
                                    type="checkbox"
                                    checked={restoreOnStartup}
                                    onChange={(e) => onToggleRestoreOnStartup(e.target.checked)}
                                />
                                <span className={styles.slider}></span>
                            </label>
                        </div>
                    </div>
                </div>
            </section>

            <section className={styles.section}>
                <h3 className={styles.sectionTitle}>Maintenance</h3>
                <div className={styles.settingItem}>
                    <div className={styles.settingIcon}>
                        <Trash2 size={17} />
                    </div>
                    <div className={styles.settingInfo}>
                        <div className={styles.settingLabel}>Clear Font Cache</div>
                        <div className={styles.settingDescription}>
                            Flush in-memory metadata & preview caches and re-scan your managed folders.
                        </div>
                    </div>
                    <div className={styles.settingAction}>
                        <button className={styles.dangerButton} onClick={onClearCache}>
                            <Trash2 size={14} />
                            <span>Clear Cache</span>
                        </button>
                    </div>
                </div>
            </section>

            <section className={styles.section}>
                <h3 className={styles.sectionTitle}>About</h3>
                <div className={styles.aboutCard}>
                    <div className={styles.versionRow}>
                        <span>Version</span>
                        <strong>{version ? `v${version}` : '...'}</strong>
                    </div>
                    <div className={styles.versionRow}>
                        <span>Build</span>
                        <strong>Release Build • windows/amd64</strong>
                    </div>
                    <p className={styles.disclaimer}>
                        <Info size={13} className={styles.disclaimerIcon} />
                        <span>
                            Fantoma F is a lightweight temporary font manager for Windows. Fonts are staged per-user without permanent system-wide installation.
                        </span>
                    </p>
                </div>
            </section>
        </div>
    )
}
