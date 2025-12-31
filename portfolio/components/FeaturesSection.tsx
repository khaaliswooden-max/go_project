import { motion } from 'framer-motion'

const features = [
  {
    icon: '◈',
    title: 'Interface-Driven Design',
    description: 'Every component follows the Interface Segregation Principle. Interfaces are defined at point of use, enabling testability and loose coupling throughout the codebase.'
  },
  {
    icon: '⬡',
    title: 'Table-Driven Testing',
    description: 'Comprehensive test coverage using Go\'s idiomatic table-driven test pattern. Every function is tested with multiple scenarios including edge cases.'
  },
  {
    icon: '◇',
    title: 'Production Patterns',
    description: 'Real-world patterns from production systems: graceful shutdown, structured logging, correlation IDs, retry with exponential backoff, and circuit breakers.'
  },
  {
    icon: '◎',
    title: 'LEARN Annotations',
    description: 'Educational comments throughout the code explain why each pattern is used, with references to academic concepts from CMU, MIT, and Stanford curricula.'
  },
  {
    icon: '⬢',
    title: 'Progressive Complexity',
    description: 'Seven phases take you from undergraduate foundations through PhD-level distributed systems, each building on concepts from previous phases.'
  },
  {
    icon: '✧',
    title: 'Production Ready',
    description: 'Not just a toy project—GLR is designed to become a core component of the Zuup platform for tool orchestration and AI agent coordination.'
  }
]

export default function FeaturesSection() {
  return (
    <section className="section">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          style={{ textAlign: 'center', marginBottom: '4rem' }}
        >
          <h2>Project Philosophy</h2>
          <p style={{ 
            color: 'var(--text-secondary)',
            fontFamily: 'var(--font-serif)',
            fontStyle: 'italic',
            fontSize: 'var(--size-lg)',
            maxWidth: '600px',
            margin: '0 auto'
          }}>
            Production code as pedagogy—learn by building real systems
          </p>
        </motion.div>

        <div className="grid grid-3">
          {features.map((feature, index) => (
            <motion.div
              key={index}
              className="card"
              initial={{ opacity: 0, y: 30 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
              style={{ textAlign: 'center' }}
            >
              <div style={{
                width: '60px',
                height: '60px',
                margin: '0 auto 1.5rem',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                fontSize: '2rem',
                color: 'var(--accent-primary)',
                background: 'var(--bg-secondary)',
                borderRadius: '12px',
                border: '1px solid var(--border-light)'
              }}>
                {feature.icon}
              </div>
              
              <h3 style={{
                fontFamily: 'var(--font-serif)',
                fontSize: 'var(--size-lg)',
                marginBottom: '0.75rem',
                color: 'var(--text-primary)'
              }}>
                {feature.title}
              </h3>
              
              <p style={{
                fontSize: 'var(--size-base)',
                color: 'var(--text-secondary)',
                lineHeight: 1.7
              }}>
                {feature.description}
              </p>
            </motion.div>
          ))}
        </div>
      </div>
    </section>
  )
}

