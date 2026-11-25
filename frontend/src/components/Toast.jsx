import { useEffect } from 'react'
import { AlertCircle, CheckCircle2, Info, AlertTriangle, X } from 'lucide-react'
import styles from './Toast.module.css'

const ICONS = {
    error: AlertCircle,
    success: CheckCircle2,
    info: Info,
    warning: AlertTriangle,
}

export default function Toast({ message, type = 'info', onClose }) {
    useEffect(() => {
        if (!message) return
        const t = setTimeout(() => onClose(), 4200)
        return () => clearTimeout(t)
    }, [message, onClose])

    if (!message) return null

    const Icon = ICONS[type] || Info

    return (
        <div className={styles.container} aria-live="polite">
            <div
                className={`${styles.toast} ${styles[type] || styles.info}`}
                role="alert"
            >
                <Icon size={16} className={styles.icon} />
                <span className={styles.message}>{message}</span>
                <button
                    className={styles.close}
                    onClick={onClose}
                    aria-label="Dismiss"
                >
                    <X size={14} />
                </button>
            </div>
        </div>
    )
}
