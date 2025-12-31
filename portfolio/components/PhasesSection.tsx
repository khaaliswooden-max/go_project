import { motion } from 'framer-motion'
import PhaseCard from './PhaseCard'

const phases = [
  {
    number: 1,
    title: 'Foundations',
    subtitle: 'Interface Design & Testing Patterns',
    description: 'Core software engineering principles: interface-based design, table-driven tests, error handling patterns, HTTP server patterns with graceful shutdown, and structured logging.',
    topics: ['Interfaces', 'Table-driven Tests', 'Error Handling', 'HTTP Patterns', 'Context'],
    difficulty: 'Undergraduate' as const,
    status: 'completed' as const,
    color: '#2d5a4a'
  },
  {
    number: 2,
    title: 'Concurrency',
    subtitle: 'Goroutines, Channels & Worker Pools',
    description: 'Go\'s concurrency model: goroutines, channels, select statements, worker pools with bounded parallelism, race condition detection, and sync primitives.',
    topics: ['Goroutines', 'Channels', 'Worker Pools', 'Race Detection', 'Semaphores'],
    difficulty: 'Undergraduate' as const,
    status: 'completed' as const,
    color: '#2d5a4a'
  },
  {
    number: 3,
    title: 'Generics',
    subtitle: 'Type Parameters & Constraints',
    description: 'Go 1.18+ generics: type parameters, constraints, generic data structures (caches, pools), and the Result type pattern for error handling.',
    topics: ['Type Parameters', 'Constraints', 'Generic Cache', 'Object Pool', 'Result[T]'],
    difficulty: 'Graduate' as const,
    status: 'completed' as const,
    color: '#4a4166'
  },
  {
    number: 4,
    title: 'Performance',
    subtitle: 'Profiling, Benchmarks & Memory Optimization',
    description: 'Performance engineering: pprof profiling, micro-benchmarking, memory allocation optimization, sync.Pool, and buffer management patterns.',
    topics: ['pprof', 'Benchmarks', 'Memory Pools', 'Allocations', 'JSON Optimization'],
    difficulty: 'Graduate' as const,
    status: 'completed' as const,
    color: '#4a4166'
  },
  {
    number: 5,
    title: 'Systems Programming',
    subtitle: 'unsafe, mmap & CGO Integration',
    description: 'Low-level systems programming: unsafe package, memory layout analysis, cross-platform memory-mapped files, and C interoperability via CGO.',
    topics: ['unsafe', 'Memory Layout', 'mmap', 'CGO', 'Cross-Platform'],
    difficulty: 'PhD' as const,
    status: 'completed' as const,
    color: '#8b4513'
  },
  {
    number: 6,
    title: 'Static Analysis',
    subtitle: 'AST Parsing & Custom Linters',
    description: 'Go tooling development: go/ast for parsing, go/types for type checking, custom linter implementation, and code generation patterns.',
    topics: ['go/ast', 'go/types', 'Linters', 'Code Generation', 'Templates'],
    difficulty: 'PhD' as const,
    status: 'completed' as const,
    color: '#8b4513'
  },
  {
    number: 7,
    title: 'Distributed Systems',
    subtitle: 'Raft Consensus & Streaming',
    description: 'Distributed systems fundamentals: simplified Raft consensus, leader election, log replication, streaming communication patterns, and distributed KV stores.',
    topics: ['Raft', 'Consensus', 'Streaming', 'Leader Election', 'Log Replication'],
    difficulty: 'PhD' as const,
    status: 'completed' as const,
    color: '#8b4513'
  }
]

export default function PhasesSection() {
  return (
    <section id="phases" className="section">
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          style={{ textAlign: 'center', marginBottom: '4rem' }}
        >
          <h2>Learning Curriculum</h2>
          <p style={{ 
            color: 'var(--text-secondary)',
            fontFamily: 'var(--font-serif)',
            fontStyle: 'italic',
            fontSize: 'var(--size-lg)',
            maxWidth: '600px',
            margin: '0 auto'
          }}>
            A seven-phase journey from undergraduate foundations to PhD-level distributed systems
          </p>

          {/* Academic difficulty legend */}
          <div style={{ 
            display: 'flex', 
            gap: '2rem', 
            justifyContent: 'center',
            marginTop: '2rem',
            flexWrap: 'wrap'
          }}>
            <LegendItem color="#2d5a4a" label="Undergraduate" />
            <LegendItem color="#4a4166" label="Graduate" />
            <LegendItem color="#8b4513" label="PhD Level" />
          </div>
        </motion.div>

        <div className="grid grid-2">
          {phases.map((phase) => (
            <PhaseCard key={phase.number} {...phase} />
          ))}
        </div>

        {/* Summary stats */}
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          style={{
            marginTop: '4rem',
            padding: '2rem',
            background: 'var(--bg-secondary)',
            borderRadius: '12px',
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
            gap: '2rem',
            textAlign: 'center'
          }}
        >
          <Stat value="7" label="Phases" />
          <Stat value="21+" label="Exercises" />
          <Stat value="50+" label="Source Files" />
          <Stat value="∞" label="Lines of Go" />
        </motion.div>
      </div>
    </section>
  )
}

function LegendItem({ color, label }: { color: string; label: string }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
      <span style={{
        width: '12px',
        height: '12px',
        borderRadius: '3px',
        background: color
      }} />
      <span style={{
        fontSize: 'var(--size-sm)',
        color: 'var(--text-muted)'
      }}>
        {label}
      </span>
    </div>
  )
}

function Stat({ value, label }: { value: string; label: string }) {
  return (
    <div>
      <div style={{
        fontFamily: 'var(--font-serif)',
        fontSize: 'var(--size-4xl)',
        fontWeight: 600,
        color: 'var(--accent-primary)',
        lineHeight: 1
      }}>
        {value}
      </div>
      <div style={{
        fontSize: 'var(--size-sm)',
        color: 'var(--text-muted)',
        marginTop: '0.5rem',
        textTransform: 'uppercase',
        letterSpacing: '0.1em'
      }}>
        {label}
      </div>
    </div>
  )
}

