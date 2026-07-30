import { resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

import { validateProject } from '../src/lib/validate.js';

const rootDir = resolve(fileURLToPath(new URL('..', import.meta.url)));
const result = await validateProject(rootDir);

console.log(JSON.stringify(result, null, 2));
