import { Html, Head, Main, NextScript } from 'next/document'

export default function Document() {
  return (
    <Html lang="en">
      <Head>
        {/* Preconnect to Google Fonts */}
        <link rel="preconnect" href="https://fonts.googleapis.com" />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" />
        
        {/* Favicon */}
        <link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>◈</text></svg>" />
        
        {/* Theme color */}
        <meta name="theme-color" content="#faf8f5" />
        
        {/* Academic/schema.org structured data */}
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{
            __html: JSON.stringify({
              '@context': 'https://schema.org',
              '@type': 'SoftwareSourceCode',
              name: 'GoLlama Runtime',
              description: 'A production-grade LLM tool orchestration runtime built in Go',
              programmingLanguage: 'Go',
              codeRepository: 'https://github.com/khaaliswooden-max/go_project',
              author: {
                '@type': 'Person',
                name: 'Author Name'
              },
              educationalLevel: 'Graduate',
              keywords: ['Go', 'Systems Programming', 'Distributed Systems', 'Raft', 'Concurrency']
            })
          }}
        />
      </Head>
      <body>
        <Main />
        <NextScript />
      </body>
    </Html>
  )
}

