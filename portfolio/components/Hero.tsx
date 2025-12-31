import { motion } from 'framer-motion'

interface HeroProps {
  title: string
  subtitle: string
  author: string
  institution?: string
}

export default function Hero({ title, subtitle, author, institution }: HeroProps) {
  return (
    <section style={{
      minHeight: '100vh',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'var(--gradient-hero)',
      position: 'relative',
      overflow: 'hidden'
    }}>
      {/* Decorative background pattern */}
      <BackgroundPattern />
      
      <div className="container" style={{ textAlign: 'center', position: 'relative', zIndex: 1 }}>
        <motion.div
          initial={{ opacity: 0, y: 40 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.8, ease: [0.25, 0.1, 0.25, 1] }}
        >
          {/* Academic ornament */}
          <div style={{ marginBottom: '2rem' }}>
            <span style={{ 
              fontSize: '3rem', 
              color: 'var(--accent-primary)',
              opacity: 0.8 
            }}>
              ❧
            </span>
          </div>

          {/* Main Title */}
          <h1 style={{
            fontFamily: 'var(--font-serif)',
            fontSize: 'clamp(2.5rem, 6vw, 4.5rem)',
            fontWeight: 400,
            lineHeight: 1.1,
            marginBottom: '1.5rem',
            color: 'var(--text-primary)',
            letterSpacing: '-0.02em'
          }}>
            {title}
          </h1>

          {/* Subtitle */}
          <p style={{
            fontFamily: 'var(--font-serif)',
            fontSize: 'clamp(1.125rem, 2vw, 1.5rem)',
            color: 'var(--text-secondary)',
            maxWidth: '700px',
            margin: '0 auto 2.5rem',
            fontStyle: 'italic',
            lineHeight: 1.6
          }}>
            {subtitle}
          </p>

          {/* Author info - academic style */}
          <div style={{
            padding: '1.5rem 2.5rem',
            display: 'inline-block',
            borderTop: '1px solid var(--border-light)',
            borderBottom: '1px solid var(--border-light)'
          }}>
            <p style={{
              fontFamily: 'var(--font-sans)',
              fontSize: 'var(--size-lg)',
              fontWeight: 500,
              color: 'var(--text-primary)',
              marginBottom: '0.25rem'
            }}>
              {author}
            </p>
            {institution && (
              <p style={{
                fontFamily: 'var(--font-sans)',
                fontSize: 'var(--size-sm)',
                color: 'var(--text-muted)',
                letterSpacing: '0.05em',
                textTransform: 'uppercase'
              }}>
                {institution}
              </p>
            )}
          </div>

          {/* CTA Buttons */}
          <motion.div 
            style={{ 
              marginTop: '3rem', 
              display: 'flex', 
              gap: '1rem', 
              justifyContent: 'center',
              flexWrap: 'wrap'
            }}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.4, duration: 0.6 }}
          >
            <a href="#phases" className="btn btn-primary">
              Explore Curriculum →
            </a>
            <a 
              href="https://github.com/khaaliswooden-max/go_project" 
              target="_blank" 
              rel="noopener noreferrer"
              className="btn btn-secondary"
            >
              View on GitHub
            </a>
          </motion.div>
        </motion.div>

        {/* Scroll indicator */}
        <motion.div
          style={{
            position: 'absolute',
            bottom: '3rem',
            left: '50%',
            transform: 'translateX(-50%)'
          }}
          initial={{ opacity: 0 }}
          animate={{ opacity: 1, y: [0, 8, 0] }}
          transition={{ 
            opacity: { delay: 1 },
            y: { repeat: Infinity, duration: 1.5, ease: 'easeInOut' }
          }}
        >
          <span style={{ 
            fontSize: '1.5rem', 
            color: 'var(--text-muted)' 
          }}>
            ↓
          </span>
        </motion.div>
      </div>
    </section>
  )
}

function BackgroundPattern() {
  return (
    <div style={{
      position: 'absolute',
      inset: 0,
      opacity: 0.03,
      backgroundImage: `url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cpath d='M30 0L60 30L30 60L0 30z' fill='none' stroke='%238b4513' stroke-width='1'/%3E%3C/svg%3E")`,
      backgroundSize: '60px 60px'
    }} />
  )
}

