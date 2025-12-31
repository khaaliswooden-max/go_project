import { motion } from 'framer-motion'
import { Highlight, themes } from 'prism-react-renderer'

interface CodeBlockProps {
  code: string
  language: string
  filename?: string
  showLineNumbers?: boolean
  highlights?: number[]
}

export default function CodeBlock({ 
  code, 
  language, 
  filename,
  showLineNumbers = true,
  highlights = []
}: CodeBlockProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.5 }}
      style={{
        borderRadius: '12px',
        overflow: 'hidden',
        boxShadow: 'var(--shadow-lg)',
        margin: '1.5rem 0'
      }}
    >
      {/* Header bar */}
      {filename && (
        <div style={{
          background: '#1a1d23',
          padding: '0.75rem 1rem',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          borderBottom: '1px solid #2a2d35'
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem' }}>
            {/* Traffic lights */}
            <div style={{ display: 'flex', gap: '0.4rem' }}>
              <span style={{ width: 12, height: 12, borderRadius: '50%', background: '#ff5f56' }} />
              <span style={{ width: 12, height: 12, borderRadius: '50%', background: '#ffbd2e' }} />
              <span style={{ width: 12, height: 12, borderRadius: '50%', background: '#27c93f' }} />
            </div>
            <span style={{
              fontFamily: 'var(--font-mono)',
              fontSize: 'var(--size-sm)',
              color: '#8a8378'
            }}>
              {filename}
            </span>
          </div>
          <span style={{
            fontSize: 'var(--size-xs)',
            padding: '0.2rem 0.5rem',
            borderRadius: '4px',
            background: '#2a2d35',
            color: '#6c685f',
            fontFamily: 'var(--font-mono)',
            textTransform: 'uppercase'
          }}>
            {language}
          </span>
        </div>
      )}

      {/* Code content */}
      <Highlight
        theme={themes.nightOwl}
        code={code.trim()}
        language={language as any}
      >
        {({ style, tokens, getLineProps, getTokenProps }) => (
          <pre style={{
            ...style,
            margin: 0,
            padding: '1.25rem',
            fontSize: 'var(--size-sm)',
            lineHeight: 1.7,
            overflow: 'auto',
            background: '#0d1017'
          }}>
            {tokens.map((line, i) => {
              const isHighlighted = highlights.includes(i + 1)
              return (
                <div 
                  key={i} 
                  {...getLineProps({ line })}
                  style={{
                    display: 'flex',
                    background: isHighlighted ? 'rgba(139, 69, 19, 0.15)' : 'transparent',
                    marginLeft: '-1.25rem',
                    marginRight: '-1.25rem',
                    paddingLeft: '1.25rem',
                    paddingRight: '1.25rem',
                    borderLeft: isHighlighted ? '3px solid var(--accent-primary)' : '3px solid transparent'
                  }}
                >
                  {showLineNumbers && (
                    <span style={{
                      display: 'inline-block',
                      width: '2.5rem',
                      textAlign: 'right',
                      paddingRight: '1rem',
                      color: '#4a4a4a',
                      userSelect: 'none'
                    }}>
                      {i + 1}
                    </span>
                  )}
                  <span>
                    {line.map((token, key) => (
                      <span key={key} {...getTokenProps({ token })} />
                    ))}
                  </span>
                </div>
              )
            })}
          </pre>
        )}
      </Highlight>
    </motion.div>
  )
}

