#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const os = require('os');

const SKILL_DIRS = ['neotex', 'neotex-init'];

function uninstall() {
  const skillsBase = path.join(os.homedir(), '.claude', 'skills');

  for (const dir of SKILL_DIRS) {
    try {
      const skillDir = path.join(skillsBase, dir);
      if (fs.existsSync(skillDir)) {
        fs.rmSync(skillDir, { recursive: true });
        console.log(`Removed ${skillDir}`);
      }
    } catch (err) {
      console.warn(`Warning: Could not remove ${dir}:`, err.message);
    }
  }
}

uninstall();
