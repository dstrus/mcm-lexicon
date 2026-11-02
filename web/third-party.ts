// thirdParty is a Vite plugin that records which npm packages end up in the
// built pages, with their licenses, for THIRD_PARTY_NOTICES.txt (made by
// cmd/notices). Only the browser's build counts: that's what MCM Lexicon ships.
import { mkdirSync, readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import type { Plugin } from 'vite';

export const thirdPartyFile = '.svelte-kit/third-party.json';

export interface ThirdPartyPackage {
	name: string;
	version: string;
	license: string; // as the package declares it, such as MIT or OFL-1.1
	text: string; // its license file, with its copyright notice
}

const licenseFile = /^(licen[cs]e|copying|ofl)(\.|-|$)/i;

export function thirdParty(): Plugin {
	return {
		name: 'lexicon-third-party',
		apply: 'build',
		generateBundle(_, bundle) {
			if (this.environment.config.consumer !== 'client') return;
			const dirs = new Set<string>();
			// Code comes from modules; fonts, pulled in by CSS, from assets.
			const sources = [...this.getModuleIds()];
			for (const out of Object.values(bundle)) {
				if (out.type === 'asset') sources.push(...(out.originalFileNames ?? []));
			}
			for (const id of sources) {
				// Asset names are relative to the project: node_modules/@fontsource/…
				const file = resolve(this.environment.config.root, id.replace(/[?#].*$/, '')).replace(/\\/g, '/');
				const at = file.lastIndexOf('/node_modules/');
				if (at < 0) continue;
				const rest = file.slice(at + '/node_modules/'.length).split('/');
				const name = rest[0]!.startsWith('@') ? `${rest[0]}/${rest[1]}` : rest[0]!;
				dirs.add(`${file.slice(0, at)}/node_modules/${name}`);
			}
			const packages: ThirdPartyPackage[] = [...dirs].map((dir) => {
				const pkg = JSON.parse(readFileSync(join(dir, 'package.json'), 'utf8'));
				const found = readdirSync(dir).find((f) => licenseFile.test(f));
				return {
					name: pkg.name,
					version: pkg.version,
					license: typeof pkg.license === 'string' ? pkg.license : '',
					text: found ? readFileSync(join(dir, found), 'utf8') : ''
				};
			});
			packages.sort((a, b) => a.name.localeCompare(b.name));
			mkdirSync(dirname(thirdPartyFile), { recursive: true });
			writeFileSync(thirdPartyFile, JSON.stringify(packages, null, '\t') + '\n');
		}
	};
}
