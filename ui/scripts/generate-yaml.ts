import { $ } from "bun";

const { stdout } = await $`cd .. && ./discuit inject-config`.quiet();
Bun.write("../ui-config.yaml", stdout);
