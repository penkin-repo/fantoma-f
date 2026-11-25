import { useEffect, useRef } from 'react'
import {
    BookOpen,
    Type,
    Terminal,
    PenTool,
    Sparkles,
    HelpCircle,
    Check,
    RotateCcw,
} from 'lucide-react'
import styles from './CategoryPicker.module.css'

const CATEGORIES = [
    { value: 'serif',      label: 'Serif',      Icon: BookOpen },
    { value: 'sans-serif', label: 'Sans-serif', Icon: Type },
    { value: 'monospace',  label: 'Monospace',  Icon: Terminal },
    { value: 'script',     label: 'Script',     Icon: PenTool },
    { value: 'display',    label: 'Display',    Icon: Sparkles },
    { value: 'other',      label: 'Other',      Icon: HelpCircle },
]

export default function CategoryPicker({ family, position, onSelect, onClose }) {
    const ref = useRef()

    useEffect(() => {
        const handler = (e) => {
            if (ref.current && !ref.current.contains(e.target)) onClose()
        }
        const keyHandler = (e) => {
            if (e.key === 'Escape') onClose()
        }
        document.addEventListener('mousedown', handler)
        document.addEventListener('keydown', keyHandler)
        return () => {
            document.removeEventListener('mousedown', handler)
            document.removeEventListener('keydown', keyHandler)
        }
    }, [onClose])

    const style = {
        top: position.y,
        left: position.x,
    }

    return (
        <div className={styles.overlay}>
            <div className={styles.popup} style={style} ref={ref}>
                <div className={styles.header}>
                    <span className={styles.title}>{family.name}</span>
                    {family.isManual && (
                        <button
                            className={styles.reset}
                            onClick={() => onSelect('')}
                            title="Reset to auto-detected"
                        >
                            <RotateCcw size={11} />
                            <span>Reset</span>
                        </button>
                    )}
                </div>
                <div className={styles.grid}>
                    {CATEGORIES.map(({ value, label, Icon }) => (
                        <button
                            key={value}
                            className={`${styles.item} ${family.category === value ? styles.current : ''}`}
                            onClick={() => onSelect(value)}
                        >
                            <Icon size={14} className={styles.icon} />
                            <span className={styles.itemLabel}>{label}</span>
                            {family.category === value && (
                                <Check size={14} className={styles.check} />
                            )}
                        </button>
                    ))}
                </div>
            </div>
        </div>
    )
}
