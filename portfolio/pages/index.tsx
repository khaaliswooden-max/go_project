import Head from 'next/head'
import Navigation from '@/components/Navigation'
import Hero from '@/components/Hero'
import FeaturesSection from '@/components/FeaturesSection'
import ResearchSection from '@/components/ResearchSection'
import PhasesSection from '@/components/PhasesSection'
import CodeShowcase from '@/components/CodeShowcase'
import Footer from '@/components/Footer'

interface HomeProps {
  theme: 'light' | 'dark'
  toggleTheme: () => void
}

export default function Home({ theme, toggleTheme }: HomeProps) {
  return (
    <>
      <Head>
        <title>GoLlama Runtime | Systems Programming Portfolio</title>
        <meta 
          name="description" 
          content="A production-grade LLM tool orchestration runtime built in Go. Teaching systems programming from undergraduate to PhD level through hands-on implementation." 
        />
        <meta name="keywords" content="Go, Golang, Systems Programming, Distributed Systems, Raft, Concurrency, Portfolio" />
        
        {/* Open Graph */}
        <meta property="og:title" content="GoLlama Runtime | Systems Programming Portfolio" />
        <meta property="og:description" content="A seven-phase systems programming curriculum from foundations to distributed consensus." />
        <meta property="og:type" content="website" />
        
        {/* Academic metadata */}
        <meta name="citation_title" content="GoLlama Runtime: A Systems Programming Curriculum" />
        <meta name="citation_author" content="Author Name" />
        <meta name="citation_publication_date" content="2024" />
      </Head>

      <Navigation theme={theme} toggleTheme={toggleTheme} />
      
      <main>
        <Hero
          title="GoLlama Runtime"
          subtitle="A production-grade LLM tool orchestration runtime built in Go—teaching systems programming from undergraduate foundations to PhD-level distributed consensus."
          author="Your Name"
          institution="Your Institution"
        />
        
        <FeaturesSection />
        
        <ResearchSection />
        
        <PhasesSection />
        
        <CodeShowcase />
      </main>
      
      <Footer />
    </>
  )
}

