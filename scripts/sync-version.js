#!/usr/bin/env node
/**
 * Syncs the version from package.json to Chart.yaml
 * Run this after `changeset version` to keep versions in sync
 */

const fs = require('fs');
const path = require('path');

const packageJsonPath = path.join(__dirname, '..', 'package.json');
const chartYamlPath = path.join(__dirname, '..', 'helm', 'eratemanager', 'Chart.yaml');

const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
const version = packageJson.version;

console.log(`Syncing version ${version} to Chart.yaml`);

// Read and update Chart.yaml
let chartContent = fs.readFileSync(chartYamlPath, 'utf8');

// Update both version and appVersion
chartContent = chartContent.replace(/^version: .+$/m, `version: ${version}`);
chartContent = chartContent.replace(/^appVersion: .+$/m, `appVersion: "${version}"`);

fs.writeFileSync(chartYamlPath, chartContent);

console.log('Version synced successfully!');
