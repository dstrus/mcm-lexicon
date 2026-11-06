// MCM Lexicon's web site, served at daveystrus.com/mcm-lexicon. Every link
// and asset path carries the /mcm-lexicon base.
import { defineConfig } from 'astro/config';

export default defineConfig({
	site: 'https://daveystrus.com',
	base: '/mcm-lexicon',
	// Code in the docs is commands and file names, set in the File font in
	// the site's own ink.
	markdown: { syntaxHighlight: false },
	devToolbar: { enabled: false }
});
