import { motion } from 'framer-motion'
import Link from 'next/link'

interface NavigationProps {
  theme: 'light' | 'dark'
  toggleTheme: () => void
}

export default function Navigation({ theme, toggleTheme }: NavigationProps) {
  return (
    <motion.nav 
      className="nav"
      initial={{ y: -20, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      transition={{ duration: 0.5 }}
    >
      <div className="container" style={{ 
        display: 'flex', 
        justifyContent: 'space-between', 
        alignItems: 'center',
        padding: '1rem 2rem'
      }}>
        <Link href="/" style={{ 
          display: 'flex', 
          alignItems: 'center', 
          gap: '0.75rem',
          textDecoration: 'none'
        }}>
          <span style={{ fontSize: '1.75rem' }}>◈</span>
          <span style={{ 
            fontFamily: 'var(--font-serif)',
            fontSize: 'var(--size-xl)',
            fontWeight: 500,
            color: 'var(--text-primary)'
          }}>
            GLR Portfolio
          </span>
        </Link>

        <div style={{ display: 'flex', alignItems: 'center', gap: '2rem' }}>
          <NavLinks />
          <ThemeToggle theme={theme} toggleTheme={toggleTheme} />
        </div>
      </div>
    </motion.nav>
  )
}

function NavLinks() {
  const links = [
    { href: '#research', label: 'Research' },
    { href: '#phases', label: 'Curriculum' },
    { href: '#code', label: 'Code' },
    { href: '#contact', label: 'Contact' },
  ]

  return (
    <div style={{ display: 'flex', gap: '1.5rem' }}>
      {links.map(link => (
        <a
          key={link.href}
          href={link.href}
          style={{
            fontSize: 'var(--size-sm)',
            fontWeight: 500,
            color: 'var(--text-secondary)',
            textDecoration: 'none',
            transition: 'color 0.2s ease',
            letterSpacing: '0.02em'
          }}
          onMouseOver={(e) => e.currentTarget.style.color = 'var(--accent-primary)'}
          onMouseOut={(e) => e.currentTarget.style.color = 'var(--text-secondary)'}
        >
          {link.label}
        </a>
      ))}
    </div>
  )
}

function ThemeToggle({ theme, toggleTheme }: { theme: 'light' | 'dark', toggleTheme: () => void }) {
  return (
    <button
      onClick={toggleTheme}
      style={{
        background: 'var(--bg-secondary)',
        border: '1px solid var(--border-light)',
        borderRadius: '8px',
        padding: '0.5rem 0.75rem',
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'center',
        gap: '0.5rem',
        fontSize: '1rem',
        transition: 'all 0.2s ease'
      }}
      aria-label="Toggle theme"
    >
      {theme === 'light' ? '☽' : '☀'}
    </button>
  )
}

