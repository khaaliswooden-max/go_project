import { motion } from 'framer-motion'

export default function Footer() {
  return (
    <footer id="contact" style={{
      background: 'var(--bg-primary)',
      borderTop: '1px solid var(--border-light)',
      padding: '4rem 0 2rem'
    }}>
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          style={{ textAlign: 'center' }}
        >
          {/* Academic ornament */}
          <div style={{ marginBottom: '2rem' }}>
            <span style={{ 
              fontSize: '2rem', 
              color: 'var(--accent-primary)',
              opacity: 0.6 
            }}>
              ❦
            </span>
          </div>

          <h2 style={{ 
            fontFamily: 'var(--font-serif)',
            fontSize: 'var(--size-2xl)',
            marginBottom: '1rem'
          }}>
            Get in Touch
          </h2>

          <p style={{
            fontFamily: 'var(--font-serif)',
            fontSize: 'var(--size-lg)',
            fontStyle: 'italic',
            color: 'var(--text-secondary)',
            marginBottom: '2rem',
            maxWidth: '500px',
            margin: '0 auto 2rem'
          }}>
            Interested in collaboration or have questions about the research?
          </p>

          {/* Contact links */}
          <div style={{ 
            display: 'flex', 
            gap: '1.5rem', 
            justifyContent: 'center',
            flexWrap: 'wrap',
            marginBottom: '3rem'
          }}>
            <ContactLink 
              href="mailto:your.email@university.edu" 
              icon="✉" 
              label="Email" 
            />
            <ContactLink 
              href="https://github.com/khaaliswooden-max/go_project" 
              icon="⌘" 
              label="GitHub" 
            />
            <ContactLink 
              href="https://linkedin.com" 
              icon="◈" 
              label="LinkedIn" 
            />
            <ContactLink 
              href="https://scholar.google.com" 
              icon="◎" 
              label="Scholar" 
            />
          </div>

          {/* Divider */}
          <div className="ornament" style={{ maxWidth: '400px', margin: '0 auto 2rem' }}>
            <span className="ornament-icon">◇</span>
          </div>

          {/* Bottom info */}
          <div style={{ 
            fontSize: 'var(--size-sm)',
            color: 'var(--text-muted)'
          }}>
            <p style={{ marginBottom: '0.5rem' }}>
              GoLlama Runtime (GLR) — A Systems Programming Curriculum
            </p>
            <p style={{ 
              fontFamily: 'var(--font-serif)',
              fontStyle: 'italic'
            }}>
              "Production-grade code as pedagogy"
            </p>
          </div>

          {/* Copyright */}
          <div style={{
            marginTop: '3rem',
            paddingTop: '2rem',
            borderTop: '1px solid var(--border-light)',
            fontSize: 'var(--size-xs)',
            color: 'var(--text-muted)',
            letterSpacing: '0.05em'
          }}>
            <p>© {new Date().getFullYear()} · MIT License · Built with Next.js</p>
          </div>
        </motion.div>
      </div>
    </footer>
  )
}

function ContactLink({ href, icon, label }: { href: string; icon: string; label: string }) {
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: '0.5rem',
        padding: '0.75rem 1.25rem',
        background: 'var(--bg-secondary)',
        border: '1px solid var(--border-light)',
        borderRadius: '8px',
        color: 'var(--text-secondary)',
        fontSize: 'var(--size-sm)',
        fontWeight: 500,
        textDecoration: 'none',
        transition: 'all 0.2s ease'
      }}
      onMouseOver={(e) => {
        e.currentTarget.style.borderColor = 'var(--accent-primary)'
        e.currentTarget.style.color = 'var(--accent-primary)'
      }}
      onMouseOut={(e) => {
        e.currentTarget.style.borderColor = 'var(--border-light)'
        e.currentTarget.style.color = 'var(--text-secondary)'
      }}
    >
      <span style={{ fontSize: '1.1rem' }}>{icon}</span>
      {label}
    </a>
  )
}

