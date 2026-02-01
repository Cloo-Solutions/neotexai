#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const os = require('os');

const SKILLS = [
  { src: 'SKILL.md', dest: 'neotex' },
  { src: 'neotex-init.md', dest: 'neotex-init', destFile: 'SKILL.md' }
];

function install() {
  const skillsBase = path.join(os.homedir(), '.claude', 'skills');

  for (const skill of SKILLS) {
    try {
      const skillDir = path.join(skillsBase, skill.dest);
      const source = path.join(__dirname, skill.src);
      const destFile = skill.destFile || skill.src;
      const dest = path.join(skillDir, destFile);

      if (!fs.existsSync(source)) {
        console.warn(`Warning: ${skill.src} not found in package, skipping`);
        continue;
      }

      // Create skill directory
      fs.mkdirSync(skillDir, { recursive: true });

      // Copy skill file
      fs.copyFileSync(source, dest);
      console.log(`Installed ${skill.src} to ${skillDir}`);
    } catch (err) {
      console.warn(`Warning: Could not install ${skill.src}:`, err.message);
    }
  }

  console.log('Neotex skills are now available in Claude Code sessions.');
}

install();
