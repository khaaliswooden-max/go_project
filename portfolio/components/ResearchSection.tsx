import { motion } from 'framer-motion'

interface Paper {
  title: string
  authors: string[]
  venue: string
  year: number
  abstract: string
  tags: string[]
  link?: string
}

const papers: Paper[] = [
  {
    title: "Interface-Based Design Patterns in Production Go Systems",
    authors: ["Author Name"],
    venue: "Software Engineering Practice",
    year: 2024,
    abstract: "An exploration of interface segregation, dependency injection, and testable design patterns as implemented in the GoLlama Runtime. This work demonstrates how academic software engineering principles (CMU 15-214, MIT 6.031) translate to production systems.",
    tags: ["Go", "Design Patterns", "Software Engineering"],
  },
  {
    title: "Bounded Concurrency: Worker Pools and Channel Patterns",
    authors: ["Author Name"],
    venue: "Concurrent Systems",
    year: 2024,
    abstract: "Implementation study of bounded parallelism using Go's concurrency primitives. Covers semaphore-based rate limiting, fan-out/fan-in patterns, and race condition prevention in high-throughput LLM request handling.",
    tags: ["Concurrency", "Go", "Worker Pools"],
  },
  {
    title: "Memory-Mapped Files and CGO: Systems Programming in Go",
    authors: ["Author Name"],
    venue: "Systems Programming",
    year: 2024,
    abstract: "Deep dive into Go's unsafe package, platform-specific memory mapping implementations, and C interoperability. Demonstrates performance tradeoffs between safe Go abstractions and low-level system calls.",
    tags: ["Systems", "CGO", "Memory Mapping"],
  },
  {
    title: "Raft Consensus for Tool Orchestration Runtimes",
    authors: ["Author Name"],
    venue: "Distributed Systems",
    year: 2024,
    abstract: "Implementation of simplified Raft consensus for distributed LLM tool execution. Covers leader election, log replication, and state machine safety in the context of AI agent coordination.",
    tags: ["Distributed Systems", "Raft", "Consensus"],
  },
]

export default function ResearchSection() {
  return (
    <section id="research" className="section" style={{ background: 'var(--bg-secondary)' }}>
      <div className="container narrow">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
        >
          <h2 style={{ textAlign: 'center' }}>Research & Publications</h2>
          <p style={{ 
            textAlign: 'center', 
            color: 'var(--text-secondary)',
            fontFamily: 'var(--font-serif)',
            fontStyle: 'italic',
            fontSize: 'var(--size-lg)',
            marginBottom: '3rem'
          }}>
            Academic contributions and technical writings
          </p>
        </motion.div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: '2rem' }}>
          {papers.map((paper, index) => (
            <PaperCard key={index} paper={paper} index={index} />
          ))}
        </div>
      </div>
    </section>
  )
}

function PaperCard({ paper, index }: { paper: Paper, index: number }) {
  return (
    <motion.article
      className="card"
      initial={{ opacity: 0, x: -30 }}
      whileInView={{ opacity: 1, x: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.5, delay: index * 0.1 }}
      style={{
        display: 'grid',
        gap: '1rem'
      }}
    >
      {/* Title */}
      <h3 style={{
        fontFamily: 'var(--font-serif)',
        fontSize: 'var(--size-xl)',
        fontWeight: 600,
        color: 'var(--text-primary)',
        lineHeight: 1.4
      }}>
        {paper.title}
      </h3>

      {/* Meta */}
      <div style={{ 
        display: 'flex', 
        alignItems: 'center', 
        gap: '1rem',
        flexWrap: 'wrap'
      }}>
        <span style={{
          fontFamily: 'var(--font-sans)',
          fontSize: 'var(--size-sm)',
          color: 'var(--text-secondary)'
        }}>
          {paper.authors.join(', ')}
        </span>
        <span style={{ color: 'var(--border-dark)' }}>•</span>
        <span style={{
          fontFamily: 'var(--font-sans)',
          fontSize: 'var(--size-sm)',
          fontStyle: 'italic',
          color: 'var(--text-muted)'
        }}>
          {paper.venue}, {paper.year}
        </span>
      </div>

      {/* Abstract */}
      <p style={{
        fontSize: 'var(--size-base)',
        color: 'var(--text-secondary)',
        lineHeight: 1.8,
        fontFamily: 'var(--font-serif)'
      }}>
        {paper.abstract}
      </p>

      {/* Tags */}
      <div style={{ display: 'flex', gap: '0.5rem', flexWrap: 'wrap', marginTop: '0.5rem' }}>
        {paper.tags.map((tag, i) => (
          <span
            key={i}
            style={{
              fontSize: 'var(--size-xs)',
              padding: '0.3rem 0.65rem',
              borderRadius: '4px',
              background: 'var(--accent-primary)',
              color: 'white',
              fontWeight: 500
            }}
          >
            {tag}
          </span>
        ))}
      </div>
    </motion.article>
  )
}

