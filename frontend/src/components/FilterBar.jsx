import { Italic, RotateCcw } from 'lucide-react'
import styles from './FilterBar.module.css'

const CATEGORIES = [
    { value: 'serif',      label: 'Serif' },
    { value: 'sans-serif', label: 'Sans' },
    { value: 'monospace',  label: 'Mono' },
    { value: 'script',     label: 'Script' },
    { value: 'display',    label: 'Display' },
    { value: 'other',      label: 'Other' },
]

const WEIGHTS = [
    { value: 100, label: 'Thin' },
    { value: 200, label: 'ExtraLight' },
    { value: 300, label: 'Light' },
    { value: 400, label: 'Regular' },
    { value: 500, label: 'Medium' },
    { value: 600, label: 'SemiBold' },
    { value: 700, label: 'Bold' },
    { value: 800, label: 'ExtraBold' },
    { value: 900, label: 'Black' },
]

export default function FilterBar({ filters, setFilters }) {
    const toggle = (key, value) => {
        setFilters(prev => {
            const current = prev[key]
            if (current === value) {
                return { ...prev, [key]: null }
            }
            return { ...prev, [key]: value }
        })
    }

    const hasActiveFilter = Boolean(filters.category || filters.weight || filters.hasItalic)

    return (
        <div className={styles.bar}>
            <div className={styles.group}>
                <span className={styles.label}>Type</span>
                {CATEGORIES.map(c => (
                    <button
                        key={c.value}
                        className={`${styles.chip} ${filters.category === c.value ? styles.active : ''}`}
                        onClick={() => toggle('category', c.value)}
                    >
                        {c.label}
                    </button>
                ))}
            </div>

            <div className={styles.divider} />

            <div className={styles.group}>
                <span className={styles.label}>Weight</span>
                {WEIGHTS.map(w => (
                    <button
                        key={w.value}
                        className={`${styles.chip} ${filters.weight === w.value ? styles.active : ''}`}
                        onClick={() => toggle('weight', w.value)}
                    >
                        {w.label}
                    </button>
                ))}
            </div>

            <div className={styles.divider} />

            <div className={styles.group}>
                <button
                    className={`${styles.chip} ${styles.iconChip} ${filters.hasItalic ? styles.active : ''}`}
                    onClick={() => setFilters(prev => ({ ...prev, hasItalic: !prev.hasItalic }))}
                >
                    <Italic size={12} />
                    <span>Italic</span>
                </button>
                {hasActiveFilter && (
                    <button
                        className={styles.reset}
                        onClick={() => setFilters({ category: null, weight: null, hasItalic: false })}
                    >
                        <RotateCcw size={12} />
                        <span>Clear</span>
                    </button>
                )}
            </div>
        </div>
    )
}
