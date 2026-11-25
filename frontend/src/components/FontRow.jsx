import { useEffect } from 'react'
import { ChevronDown, ChevronRight, Check } from 'lucide-react'
import { fontFaceId, ensureFontLoaded } from '../utils/fontLoader'
import styles from './FontRow.module.css'

const DEFAULT_PREVIEW = 'The quick brown fox jumps over the lazy dog'

const CATEGORY_LABELS = {
    'serif': 'Serif',
    'sans-serif': 'Sans',
    'monospace': 'Mono',
    'script': 'Script',
    'display': 'Display',
    'other': 'Other',
}

function StyleSubRow({ familyName, styleItem, previewText, fontSize, onToggleStyle, isToggling }) {
    const styleFaceId = fontFaceId(styleItem.filePath)

    useEffect(() => {
        ensureFontLoaded(styleFaceId, styleItem.filePath)
    }, [styleFaceId, styleItem.filePath])

    const subPreviewSize = Math.min(Math.max(16, Math.round(fontSize * 0.78)), 42)

    return (
        <div className={`${styles.styleRow} ${styleItem.isActive ? styles.styleRowActive : ''}`}>
            <div className={styles.styleInfo}>
                <span className={styles.styleName}>{styleItem.style}</span>
                <span className={styles.formatTag}>{styleItem.format}</span>
            </div>
            <div
                className={styles.stylePreview}
                style={{
                    fontFamily: `'${styleFaceId}', sans-serif`,
                    fontSize: `${subPreviewSize}px`,
                }}
            >
                {previewText || DEFAULT_PREVIEW}
            </div>
            <div className={styles.styleActions}>
                <button
                    className={`${styles.styleToggle} ${styleItem.isActive ? styles.isActive : ''}`}
                    onClick={() => onToggleStyle(familyName, styleItem)}
                    disabled={isToggling}
                >
                    {isToggling ? (
                        <div className="spinner" />
                    ) : styleItem.isActive ? (
                        <>
                            <Check size={12} strokeWidth={2.6} />
                            <span>On</span>
                        </>
                    ) : (
                        <span>Off</span>
                    )}
                </button>
            </div>
        </div>
    )
}

export default function FontRow({
    family,
    onToggle,
    onToggleStyle,
    onCategoryClick,
    previewText,
    fontSize,
    isToggling,
    activeActions,
    expanded,
    onToggleExpand,
}) {
    const familyId = fontFaceId(family.regularPath || family.name)

    useEffect(() => {
        ensureFontLoaded(familyId, family.regularPath)
    }, [familyId, family.regularPath])

    const hasMultipleStyles = family.stylesCount > 1

    return (
        <div className={`${styles.wrapper} ${family.isActive ? styles.activeWrapper : ''}`}>
            <div className={styles.row}>
                <div className={styles.info}>
                    <div className={styles.name} title={family.name}>{family.name}</div>
                    <div className={styles.metaRow}>
                        {hasMultipleStyles ? (
                            <button
                                className={styles.stylesBtn}
                                onClick={onToggleExpand}
                                title={expanded ? 'Hide styles' : 'Show individual styles'}
                            >
                                {expanded ? <ChevronDown size={12} /> : <ChevronRight size={12} />}
                                <span>{family.stylesCount} styles</span>
                            </button>
                        ) : (
                            <span className={styles.meta}>1 style</span>
                        )}
                        <button
                            className={`${styles.categoryBadge} ${family.isManual ? styles.categoryManual : ''}`}
                            onClick={(e) => {
                                e.stopPropagation()
                                onCategoryClick && onCategoryClick(family, e)
                            }}
                            title={family.isManual ? 'Manually set — click to change' : 'Auto-detected — click to override'}
                        >
                            {CATEGORY_LABELS[family.category] || 'Other'}
                        </button>
                    </div>
                </div>

                <div
                    className={styles.preview}
                    style={{
                        fontFamily: `'${familyId}', sans-serif`,
                        fontSize: `${fontSize}px`,
                    }}
                >
                    {previewText || DEFAULT_PREVIEW}
                </div>

                <div className={styles.actions}>
                    <button
                        className={`${styles.toggle} ${family.isFullyActive ? styles.isActive : (family.isActive ? styles.isPartial : '')}`}
                        onClick={() => onToggle(family)}
                        disabled={isToggling}
                    >
                        {isToggling ? (
                            <div className="spinner" />
                        ) : family.isFullyActive ? (
                            <>
                                <Check size={13} strokeWidth={2.6} />
                                <span>Active</span>
                            </>
                        ) : family.isActive ? (
                            'Partial'
                        ) : (
                            'Activate'
                        )}
                    </button>
                </div>
            </div>

            {expanded && hasMultipleStyles && (
                <div className={styles.stylesDrawer}>
                    {family.styles.map((s) => (
                        <StyleSubRow
                            key={s.filePath}
                            familyName={family.name}
                            styleItem={s}
                            previewText={previewText}
                            fontSize={fontSize}
                            onToggleStyle={onToggleStyle}
                            isToggling={Boolean(activeActions?.[s.filePath])}
                        />
                    ))}
                </div>
            )}
        </div>
    )
}
