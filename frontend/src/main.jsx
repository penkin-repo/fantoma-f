import React from 'react'
import { createRoot } from 'react-dom/client'
import './style.css'
import App from './App'

class ErrorBoundary extends React.Component {
    constructor(props) {
        super(props)
        this.state = { error: null }
    }
    static getDerivedStateFromError(error) {
        return { error }
    }
    render() {
        if (this.state.error) {
            return (
                <div style={{ padding: 32, fontFamily: 'monospace', fontSize: 13, color: '#c00' }}>
                    <b>App crashed:</b>
                    <pre style={{ whiteSpace: 'pre-wrap', marginTop: 12 }}>
                        {this.state.error?.message}
                        {'\n\n'}
                        {this.state.error?.stack}
                    </pre>
                </div>
            )
        }
        return this.props.children
    }
}

const container = document.getElementById('root')
const root = createRoot(container)

root.render(
    <ErrorBoundary>
        <App />
    </ErrorBoundary>
)
