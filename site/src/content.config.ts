// The user documentation: one Markdown file per page in src/content/docs,
// listed in the docs' column by `order`.
import { defineCollection } from 'astro:content';
import { glob } from 'astro/loaders';
import { z } from 'astro/zod';

const docs = defineCollection({
	loader: glob({ pattern: '*.md', base: './src/content/docs' }),
	schema: z.object({
		title: z.string(),
		description: z.string(),
		order: z.number()
	})
});

export const collections = { docs };
