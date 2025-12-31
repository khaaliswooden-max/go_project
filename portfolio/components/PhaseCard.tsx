import { motion } from 'framer-motion'

interface PhaseCardProps {
  number: number
  title: string
  subtitle: string
  description: string
  topics: string[]
  difficulty: 'Undergraduate' | 'Graduate' | 'PhD'
  status: 'completed' | 'in-progress' | 'upcoming'
  color: string
}

const difficultyColors = {
  'Undergraduate': '#2d5a4a',
  'Graduate': '#4a4166',
  'PhD': '#8b4513'
}

export default function PhaseCard({ 
  number, 
  title, 
  subtitle,
  description, 
  topics, 
  difficulty,
  status,
  color 
}: PhaseCardProps) {
  return (
    <motion.div
      className="card"
      style={{
        position: 'relative',
        overflow: 'hidden'
      }}
      initial={{ opacity: 0, y: 30 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true, margin: '-50px' }}
      transition={{ duration: 0.5 }}
      whileHover={{ y: -4 }}
    >
      {/* Phase number watermark */}
      <div style={{
        position: 'absolute',
        top: '-20px',
        right: '-10px',
        fontFamily: 'var(--font-serif)',
        fontSize: '8rem',
        fontWeight: 700,
        color: color,
        opacity: 0.06,
        lineHeight: 1,
        pointerEvents: 'none'
      }}>
        {number}
      </div>

      {/* Header */}
      <div style={{ 
        display: 'flex', 
        alignItems: 'flex-start', 
        justifyContent: 'space-between',
        marginBottom: '1rem'
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
          <span style={{
            fontFamily: 'var(--font-serif)',
            fontSize: 'var(--size-3xl)',
            fontWeight: 600,
            color: color,
            lineHeight: 1
          }}>
            {number.toString().padStart(2, '0')}
          </span>
          <div style={{
            width: '40px',
            height: '2px',
            background: color
          }} />
        </div>
        
        <div style={{ display: 'flex', gap: '0.5rem' }}>
          <span style={{
            fontSize: 'var(--size-xs)',
            padding: '0.25rem 0.6rem',
            borderRadius: '12px',
            background: difficultyColors[difficulty],
            color: 'white',
            fontWeight: 600,
            letterSpacing: '0.05em'
          }}>
            {difficulty}
          </span>
          <StatusBadge status={status} />
        </div>
      </div>

      {/* Title */}
      <h3 style={{
        fontFamily: 'var(--font-serif)',
        fontSize: 'var(--size-2xl)',
        fontWeight: 500,
        marginBottom: '0.25rem',
        color: 'var(--text-primary)'
      }}>
        {title}
      </h3>

      {/* Subtitle */}
      <p style={{
        fontFamily: 'var(--font-serif)',
        fontSize: 'var(--size-base)',
        fontStyle: 'italic',
        color: 'var(--text-muted)',
        marginBottom: '1rem'
      }}>
        {subtitle}
      </p>

      {/* Description */}
      <p style={{
        fontSize: 'var(--size-base)',
        color: 'var(--text-secondary)',
        lineHeight: 1.7,
        marginBottom: '1.5rem'
      }}>
        {description}
      </p>

      {/* Topics */}
      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem' }}>
        {topics.map((topic, i) => (
          <span
            key={i}
            style={{
              fontSize: 'var(--size-xs)',
              padding: '0.35rem 0.7rem',
              borderRadius: '6px',
              background: 'var(--bg-secondary)',
              color: 'var(--text-secondary)',
              border: '1px solid var(--border-light)',
              fontFamily: 'var(--font-mono)'
            }}
          >
            {topic}
          </span>
        ))}
      </div>

      {/* Learn more link */}
      <div style={{ marginTop: '1.5rem', paddingTop: '1rem', borderTop: '1px solid var(--border-light)' }}>
        <a 
          href={`#phase-${number}`}
          style={{
            fontSize: 'var(--size-sm)',
            fontWeight: 600,
            color: color,
            display: 'flex',
            alignItems: 'center',
            gap: '0.5rem'
          }}
        >
          Explore Phase {number} →
        </a>
      </div>
    </motion.div>
  )
}

function StatusBadge({ status }: { status: 'completed' | 'in-progress' | 'upcoming' }) {
  const styles = {
    'completed': { bg: '#2d5a4a', text: '✓ Complete' },
    'in-progress': { bg: '#8b4513', text: '◉ Active' },
    'upcoming': { bg: 'var(--text-muted)', text: '○ Upcoming' }
  }

  const style = styles[status]

  return (
    <span style={{
      fontSize: 'var(--size-xs)',
      padding: '0.25rem 0.6rem',
      borderRadius: '12px',
      background: style.bg,
      color: 'white',
      fontWeight: 500
    }}>
      {style.text}
    </span>
  )
}

