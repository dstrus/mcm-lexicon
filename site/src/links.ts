// Where the site sends people outside itself.
export const repository = 'https://github.com/dstrus/mcm-lexicon';
export const releases = `${repository}/releases/latest`;
export const issues = `${repository}/issues`;
export const license = `${repository}/blob/main/LICENSE`;
export const developing = `${repository}/blob/main/DEVELOPING.md`;
export const mcmRecorder = 'https://www.nexusmods.com/skyrimspecialedition/mods/61719';

// Releases are Windows only, attached to each GitHub release under the same
// name (see `make release`), so this always fetches the latest release's.
export const download = `${releases}/download/MCMLexicon-windows-x64.zip`;
