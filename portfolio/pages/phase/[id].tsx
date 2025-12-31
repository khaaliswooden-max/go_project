import { GetStaticPaths, GetStaticProps } from 'next'
import Head from 'next/head'
import Link from 'next/link'
import { motion } from 'framer-motion'
import Navigation from '@/components/Navigation'
import CodeBlock from '@/components/CodeBlock'
import Footer from '@/components/Footer'

interface PhaseData {
  id: number
  title: string
  subtitle: string
  difficulty: string
  description: string
  concepts: {
    title: string
    description: string
    code?: string
    filename?: string
  }[]
  exercises: {
    number: string
    title: string
    description: string
    status: 'completed' | 'in-progress' | 'upcoming'
  }[]
  references: {
    title: string
    link: string
    type: 'paper' | 'course' | 'book'
  }[]
}

const phases: Record<string, PhaseData> = {
  '1': {
    id: 1,
    title: 'Foundations',
    subtitle: 'Interface Design & Testing Patterns',
    difficulty: 'Undergraduate',
    description: 'This phase covers the foundational patterns every Go developer needs: interface-based design, table-driven testing, error handling, HTTP server patterns, and structured logging.',
    concepts: [
      {
        title: 'Interface-Based Design',
        description: 'Interfaces in Go define behavior, not data. They enable dependency injection and testability by defining contracts at the point of use.',
        filename: 'internal/ollama/client.go',
        code: `// Define interfaces at point of USE, not implementation
type Client interface {
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
}

// Return interface from constructor
func New(cfg Config) (Client, error) {
    return &client{baseURL: cfg.BaseURL}, nil
}`
      },
      {
        title: 'Table-Driven Tests',
        description: 'A single test function with a slice of test cases that covers all code paths—the idiomatic Go testing pattern.',
        filename: 'internal/ollama/client_test.go',
        code: `tests := []struct {
    name    string
    input   Request
    want    Response
    wantErr bool
}{
    {name: "valid request", input: validReq, want: validResp},
    {name: "missing field", input: badReq, wantErr: true},
}

for _, tc := range tests {
    t.Run(tc.name, func(t *testing.T) {
        got, err := function(tc.input)
        if (err != nil) != tc.wantErr {
            t.Errorf("error = %v, wantErr %v", err, tc.wantErr)
        }
    })
}`
      }
    ],
    exercises: [
      { number: '1.1', title: 'Retry with Exponential Backoff', description: 'Implement retry logic with configurable backoff', status: 'completed' },
      { number: '1.2', title: 'Logging Middleware', description: 'Add correlation IDs and request logging', status: 'completed' }
    ],
    references: [
      { title: 'CMU 15-214: Software Engineering', link: 'https://www.cs.cmu.edu/~charlie/courses/15-214/', type: 'course' },
      { title: 'MIT 6.031: Software Construction', link: 'https://web.mit.edu/6.031/', type: 'course' }
    ]
  },
  // Add more phases as needed...
}

interface PhasePageProps {
  phase: PhaseData
  theme: 'light' | 'dark'
  toggleTheme: () => void
}

export default function PhasePage({ phase, theme, toggleTheme }: PhasePageProps) {
  return (
    <>
      <Head>
        <title>Phase {phase.id}: {phase.title} | GLR Portfolio</title>
        <meta name="description" content={phase.description} />
      </Head>

      <Navigation theme={theme} toggleTheme={toggleTheme} />

      <main style={{ paddingTop: '80px' }}>
        {/* Hero */}
        <section style={{
          padding: '4rem 0',
          background: 'var(--gradient-hero)',
          borderBottom: '1px solid var(--border-light)'
        }}>
          <div className="container narrow">
            <motion.div
              initial={{ opacity: 0, y: 30 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.6 }}
            >
              <Link 
                href="/#phases"
                style={{
                  display: 'inline-flex',
                  alignItems: 'center',
                  gap: '0.5rem',
                  fontSize: 'var(--size-sm)',
                  color: 'var(--text-muted)',
                  marginBottom: '1.5rem'
                }}
              >
                ← Back to Curriculum
              </Link>

              <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', marginBottom: '1rem' }}>
                <span style={{
                  fontFamily: 'var(--font-serif)',
                  fontSize: 'var(--size-4xl)',
                  fontWeight: 600,
                  color: 'var(--accent-primary)'
                }}>
                  {phase.id.toString().padStart(2, '0')}
                </span>
                <span style={{
                  padding: '0.35rem 0.85rem',
                  background: 'var(--accent-primary)',
                  color: 'white',
                  borderRadius: '20px',
                  fontSize: 'var(--size-xs)',
                  fontWeight: 600,
                  textTransform: 'uppercase',
                  letterSpacing: '0.08em'
                }}>
                  {phase.difficulty}
                </span>
              </div>

              <h1 style={{ marginBottom: '0.5rem' }}>{phase.title}</h1>
              <p style={{
                fontFamily: 'var(--font-serif)',
                fontSize: 'var(--size-xl)',
                fontStyle: 'italic',
                color: 'var(--text-secondary)'
              }}>
                {phase.subtitle}
              </p>
            </motion.div>
          </div>
        </section>

        {/* Overview */}
        <section className="section">
          <div className="container narrow">
            <h2>Overview</h2>
            <p style={{ fontSize: 'var(--size-lg)', lineHeight: 1.8 }}>
              {phase.description}
            </p>
          </div>
        </section>

        {/* Concepts */}
        <section className="section" style={{ background: 'var(--bg-secondary)' }}>
          <div className="container narrow">
            <h2>Key Concepts</h2>
            {phase.concepts.map((concept, i) => (
              <motion.div
                key={i}
                initial={{ opacity: 0, y: 20 }}
                whileInView={{ opacity: 1, y: 0 }}
                viewport={{ once: true }}
                transition={{ duration: 0.5, delay: i * 0.1 }}
                style={{ marginBottom: '3rem' }}
              >
                <h3 style={{
                  fontFamily: 'var(--font-serif)',
                  fontSize: 'var(--size-xl)',
                  marginBottom: '0.75rem'
                }}>
                  {concept.title}
                </h3>
                <p style={{ marginBottom: '1rem' }}>{concept.description}</p>
                {concept.code && (
                  <CodeBlock
                    code={concept.code}
                    language="go"
                    filename={concept.filename}
                  />
                )}
              </motion.div>
            ))}
          </div>
        </section>

        {/* Exercises */}
        <section className="section">
          <div className="container narrow">
            <h2>Exercises</h2>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '1rem' }}>
              {phase.exercises.map((exercise, i) => (
                <div
                  key={i}
                  className="card"
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between'
                  }}
                >
                  <div>
                    <span style={{
                      fontFamily: 'var(--font-mono)',
                      fontSize: 'var(--size-sm)',
                      color: 'var(--accent-primary)',
                      marginRight: '1rem'
                    }}>
                      {exercise.number}
                    </span>
                    <strong>{exercise.title}</strong>
                    <p style={{
                      fontSize: 'var(--size-sm)',
                      color: 'var(--text-muted)',
                      marginTop: '0.25rem',
                      marginBottom: 0
                    }}>
                      {exercise.description}
                    </p>
                  </div>
                  <span style={{
                    padding: '0.3rem 0.7rem',
                    borderRadius: '12px',
                    fontSize: 'var(--size-xs)',
                    fontWeight: 600,
                    background: exercise.status === 'completed' ? '#2d5a4a' : 'var(--text-muted)',
                    color: 'white'
                  }}>
                    {exercise.status === 'completed' ? '✓ Complete' : 'Pending'}
                  </span>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* References */}
        <section className="section" style={{ background: 'var(--bg-secondary)' }}>
          <div className="container narrow">
            <h2>References</h2>
            <div style={{ display: 'flex', flexDirection: 'column', gap: '0.75rem' }}>
              {phase.references.map((ref, i) => (
                <a
                  key={i}
                  href={ref.link}
                  target="_blank"
                  rel="noopener noreferrer"
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    gap: '1rem',
                    padding: '1rem',
                    background: 'var(--bg-primary)',
                    border: '1px solid var(--border-light)',
                    borderRadius: '8px',
                    textDecoration: 'none',
                    transition: 'border-color 0.2s'
                  }}
                >
                  <span style={{
                    fontSize: '1.5rem',
                    opacity: 0.6
                  }}>
                    {ref.type === 'paper' ? '📄' : ref.type === 'course' ? '🎓' : '📚'}
                  </span>
                  <span style={{ color: 'var(--text-primary)' }}>{ref.title}</span>
                  <span style={{ marginLeft: 'auto', color: 'var(--text-muted)' }}>→</span>
                </a>
              ))}
            </div>
          </div>
        </section>
      </main>

      <Footer />
    </>
  )
}

export const getStaticPaths: GetStaticPaths = async () => {
  return {
    paths: Object.keys(phases).map(id => ({ params: { id } })),
    fallback: false
  }
}

export const getStaticProps: GetStaticProps = async ({ params }) => {
  const phase = phases[params?.id as string]
  return { props: { phase } }
}

