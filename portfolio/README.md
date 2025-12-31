# GLR Portfolio

A scholarly academic portfolio website for showcasing the GoLlama Runtime (GLR) systems programming curriculum.

## Overview

This portfolio is designed for Computer Science PhD students and researchers to present their work in a professional, academic format. It features:

- **Scholarly Aesthetic**: Clean typography with Cormorant Garamond for headings and Source Sans for body text
- **Dark/Light Themes**: Toggle between parchment-toned light mode and deep dark mode
- **Code Highlighting**: Syntax-highlighted code blocks with line numbers
- **Research Section**: Academic paper-style publication listings
- **Phase Navigation**: Detailed breakdown of the 7-phase curriculum
- **Responsive Design**: Works on desktop, tablet, and mobile

## Quick Start

```bash
cd portfolio

# Install dependencies
npm install

# Start development server
npm run dev

# Build for production
npm run build

# Export static site (for GitHub Pages)
npm run export
```

## Deployment

### GitHub Pages

1. Run `npm run export` to generate static files in `out/`
2. Push the `out/` directory to a `gh-pages` branch
3. Enable GitHub Pages in repository settings

### Vercel

1. Connect your repository to Vercel
2. It will auto-detect Next.js and deploy

### Netlify

1. Set build command: `npm run build`
2. Set publish directory: `out`
3. Deploy

## Customization

### Personal Information

Edit `pages/index.tsx` to update:
- Your name
- Institution
- Contact links

### Research Papers

Edit `components/ResearchSection.tsx` to add your publications.

### Phase Details

Add more detailed phase pages by extending the `phases` object in `pages/phase/[id].tsx`.

### Colors & Theming

Modify CSS variables in `styles/globals.css`:
- `--accent-primary`: Main accent color (default: sienna brown)
- `--font-serif`: Heading font
- `--font-sans`: Body font
- `--font-mono`: Code font

## Project Structure

```
portfolio/
├── components/
│   ├── Navigation.tsx    # Header navigation
│   ├── Hero.tsx          # Landing hero section
│   ├── PhaseCard.tsx     # Phase curriculum cards
│   ├── CodeBlock.tsx     # Syntax-highlighted code
│   ├── ResearchSection.tsx
│   ├── PhasesSection.tsx
│   ├── CodeShowcase.tsx
│   ├── FeaturesSection.tsx
│   └── Footer.tsx
├── pages/
│   ├── _app.tsx          # App wrapper with theme
│   ├── _document.tsx     # HTML document
│   ├── index.tsx         # Main landing page
│   └── phase/[id].tsx    # Dynamic phase pages
├── styles/
│   └── globals.css       # Global styles & CSS variables
├── package.json
├── tsconfig.json
└── next.config.js
```

## Tech Stack

- **Framework**: Next.js 14 with static export
- **Styling**: CSS with CSS Variables (no framework)
- **Animation**: Framer Motion
- **Code Highlighting**: Prism React Renderer
- **Fonts**: Google Fonts (Cormorant Garamond, Source Sans 3, JetBrains Mono)

## Academic Features

- **Citation-ready metadata**: Open Graph and schema.org structured data
- **Accessible**: Semantic HTML, keyboard navigation, screen reader friendly
- **Print-friendly**: Styles degrade gracefully for printing

## License

MIT

