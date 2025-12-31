import { motion } from 'framer-motion'
import { useState } from 'react'
import CodeBlock from './CodeBlock'

const codeExamples = [
  {
    id: 'interfaces',
    title: 'Interface Design',
    description: 'Interfaces defined at point of use, enabling testability and loose coupling.',
    filename: 'internal/ollama/client.go',
    language: 'go',
    code: `// LEARN: Define interfaces at point of USE, not implementation
// This enables dependency injection and easy mocking

// Client abstracts Ollama API operations
type Client interface {
    // Generate performs text generation
    Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
    
    // Chat performs multi-turn conversation
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    
    // ListModels returns available models
    ListModels(ctx context.Context) ([]string, error)
}

// New returns a Client implementation
// Note: returns interface, not concrete type
func New(cfg Config) (Client, error) {
    if cfg.BaseURL == "" {
        return nil, ErrInvalidConfig
    }
    
    return &client{
        baseURL:    cfg.BaseURL,
        httpClient: &http.Client{Timeout: cfg.Timeout},
    }, nil
}`
  },
  {
    id: 'concurrency',
    title: 'Worker Pool',
    description: 'Bounded parallelism with graceful shutdown and context cancellation.',
    filename: 'internal/worker/pool.go',
    language: 'go',
    code: `// LEARN: Worker pools prevent resource exhaustion
// by limiting concurrent operations

type Pool struct {
    workers   int
    tasks     chan Task
    results   chan Result
    sem       *Semaphore
    wg        sync.WaitGroup
    ctx       context.Context
    cancel    context.CancelFunc
}

func (p *Pool) Submit(task Task) error {
    select {
    case <-p.ctx.Done():
        return ErrPoolShutdown
    case p.tasks <- task:
        return nil
    }
}

func (p *Pool) worker() {
    defer p.wg.Done()
    
    for {
        select {
        case <-p.ctx.Done():
            return
        case task := <-p.tasks:
            // Acquire semaphore slot
            p.sem.Acquire(p.ctx)
            result := task.Execute(p.ctx)
            p.sem.Release()
            
            p.results <- result
        }
    }
}`
  },
  {
    id: 'generics',
    title: 'Generic Cache',
    description: 'Type-safe LRU cache using Go generics with TTL support.',
    filename: 'pkg/generic/cache.go',
    language: 'go',
    code: `// LEARN: Generics enable type-safe reusable containers
// K must be comparable (for map keys)

type Cache[K comparable, V any] struct {
    data     map[K]*entry[V]
    order    *list[K]
    capacity int
    ttl      time.Duration
    mu       sync.RWMutex
}

type entry[V any] struct {
    value     V
    expiresAt time.Time
}

func (c *Cache[K, V]) Get(key K) (V, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    e, ok := c.data[key]
    if !ok || time.Now().After(e.expiresAt) {
        var zero V
        return zero, false
    }
    
    return e.value, true
}

func (c *Cache[K, V]) Set(key K, value V) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.data[key] = &entry[V]{
        value:     value,
        expiresAt: time.Now().Add(c.ttl),
    }
    c.evictIfNeeded()
}`
  },
  {
    id: 'systems',
    title: 'Memory-Mapped Files',
    description: 'Cross-platform mmap implementation for efficient file access.',
    filename: 'pkg/systems/mmap.go',
    language: 'go',
    code: `// LEARN: Memory mapping trades complexity for performance
// OS handles paging data in/out automatically

type MappedFile interface {
    Data() []byte
    Close() error
    Size() int
    Sync() error
}

// Unix implementation using syscall.Mmap
func openMappedUnix(path string) (*unixMapped, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    
    info, _ := f.Stat()
    size := int(info.Size())
    
    data, err := syscall.Mmap(
        int(f.Fd()),
        0,
        size,
        syscall.PROT_READ,
        syscall.MAP_SHARED,
    )
    if err != nil {
        f.Close()
        return nil, fmt.Errorf("mmap failed: %w", err)
    }
    
    return &unixMapped{file: f, data: data}, nil
}`
  },
  {
    id: 'distributed',
    title: 'Raft Leader Election',
    description: 'Simplified Raft consensus for distributed coordination.',
    filename: 'pkg/distributed/raft.go',
    language: 'go',
    code: `// LEARN: Raft provides consensus through leader election
// Higher term = more recent leadership

func (n *RaftNode) startElection() {
    n.mu.Lock()
    n.currentTerm++       // Increment term
    n.state = Candidate   // Become candidate  
    n.votedFor = n.id     // Vote for self
    n.mu.Unlock()
    
    var votes int32 = 1   // Already have own vote
    
    // Request votes from all peers in parallel
    for _, peer := range n.peers {
        go func(p Peer) {
            reply, err := p.RequestVote(RequestVoteArgs{
                Term:         n.currentTerm,
                CandidateID:  n.id,
                LastLogIndex: n.log.LastIndex(),
                LastLogTerm:  n.log.LastTerm(),
            })
            
            if err != nil || !reply.VoteGranted {
                return
            }
            
            // LEARN: Majority vote wins election
            if atomic.AddInt32(&votes, 1) > int32(len(n.peers)/2) {
                n.becomeLeader()
            }
        }(peer)
    }
}`
  }
]

export default function CodeShowcase() {
  const [activeTab, setActiveTab] = useState(codeExamples[0].id)
  const activeExample = codeExamples.find(e => e.id === activeTab)!

  return (
    <section id="code" className="section" style={{ background: 'var(--bg-secondary)' }}>
      <div className="container">
        <motion.div
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
          transition={{ duration: 0.6 }}
          style={{ textAlign: 'center', marginBottom: '3rem' }}
        >
          <h2>Code Samples</h2>
          <p style={{ 
            color: 'var(--text-secondary)',
            fontFamily: 'var(--font-serif)',
            fontStyle: 'italic',
            fontSize: 'var(--size-lg)',
            maxWidth: '600px',
            margin: '0 auto'
          }}>
            Explore patterns and implementations from across the curriculum
          </p>
        </motion.div>

        <div style={{ 
          display: 'grid', 
          gridTemplateColumns: '280px 1fr',
          gap: '2rem',
          alignItems: 'start'
        }}>
          {/* Tab sidebar */}
          <div className="toc">
            <div className="toc-title">Examples</div>
            <ul className="toc-list">
              {codeExamples.map((example) => (
                <li key={example.id} className="toc-item">
                  <button
                    onClick={() => setActiveTab(example.id)}
                    className={`toc-link ${activeTab === example.id ? 'active' : ''}`}
                    style={{
                      background: 'none',
                      border: 'none',
                      cursor: 'pointer',
                      width: '100%',
                      textAlign: 'left',
                      fontFamily: 'var(--font-sans)'
                    }}
                  >
                    {example.title}
                  </button>
                </li>
              ))}
            </ul>
          </div>

          {/* Code display */}
          <motion.div
            key={activeTab}
            initial={{ opacity: 0, x: 20 }}
            animate={{ opacity: 1, x: 0 }}
            transition={{ duration: 0.3 }}
          >
            <div style={{ marginBottom: '1rem' }}>
              <h3 style={{
                fontFamily: 'var(--font-serif)',
                fontSize: 'var(--size-xl)',
                marginBottom: '0.5rem'
              }}>
                {activeExample.title}
              </h3>
              <p style={{
                color: 'var(--text-secondary)',
                fontSize: 'var(--size-base)'
              }}>
                {activeExample.description}
              </p>
            </div>
            
            <CodeBlock
              code={activeExample.code}
              language={activeExample.language}
              filename={activeExample.filename}
              highlights={[]}
            />
          </motion.div>
        </div>
      </div>
    </section>
  )
}

