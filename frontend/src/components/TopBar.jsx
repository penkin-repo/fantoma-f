import { CheckCircle2, ALargeSmall } from 'lucide-react'
import styles from './TopBar.module.css'

export default function TopBar({
    previewText,
    setPreviewText,
    fontSize,
    setFontSize,
    fontCount,
    showActiveOnly,
    setShowActiveOnly,
}) {
    return (
        <header className={styles.topBar}>
            <div className={styles.previewControl}>
                <input
                    type="text"
                    className={styles.previewInput}
                    placeholder="Type something to preview fonts..."
                    value={previewText}
                    onChange={(e) => setPreviewText(e.target.value)}
                />
            </div>

            <div className={styles.rightControls}>
                <div className={styles.sliderGroup}>
                    <ALargeSmall size={15} className={styles.sliderIcon} />
                    <input
                        type="range"
                        min="12"
                        max="120"
                        className={styles.slider}
                        value={fontSize}
                        onChange={(e) => setFontSize(parseInt(e.target.value, 10))}
                    />
                    <span className={styles.sliderLabel}>{fontSize}px</span>
                </div>

                <button
                    className={`${styles.filterBtn} ${showActiveOnly ? styles.filterActive : ''}`}
                    onClick={() => setShowActiveOnly(!showActiveOnly)}
                    title="Show active fonts only"
                >
                    <CheckCircle2 size={14} />
                    <span>Active only</span>
                </button>

                <div className={styles.stats}>
                    {fontCount} {fontCount === 1 ? 'family' : 'families'}
                </div>
            </div>
        </header>
    )
}
