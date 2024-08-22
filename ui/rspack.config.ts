import fs from "node:fs";
import path from "node:path";
import { defineConfig } from "@rspack/cli";
import {
  type DevTool,
  type RspackPluginFunction,
  type RspackPluginInstance,
  type RuleSetRule,
  rspack,
} from "@rspack/core";
import CompressionPlugin from "compression-webpack-plugin";
import NodePolyfillPlugin from "node-polyfill-webpack-plugin";
import YAML from "yaml";

function makeid(length: number) {
  let result = "";
  const chars = "abcdefghijklmnopqrstuvwxyz0123456789";
  const charactersLength = chars.length;
  let counter = 0;
  while (counter < length) {
    result += chars.charAt(Math.floor(Math.random() * charactersLength));
    counter += 1;
  }
  return result;
}

function readYamlConfigFile() {
  const file = fs.readFileSync("../ui-config.yaml", "utf-8");
  const preConfig = YAML.parse(file);
  const allowedKeys = [
    "siteName",
    "captchaSiteKey",
    "emailContact",
    "facebookURL",
    "twitterURL",
    "instagramURL",
    "discordURL",
    "githubURL",
    "substackURL",
    "disableImagePosts",
    "disableForumCreation",
    "forumCreationReqPoints",
    "defaultFeedSort",
    "maxImagesPerPost",
  ];

  const config: { [key: string]: string } = {};
  for (const key in preConfig) {
    if (allowedKeys.includes(key)) {
      config[key] = preConfig[key];
    }
  }
  if (!config.defaultFeedSort) {
    config.defaultFeedSort = "hot";
  }
  config.communityPrefix = "+"; // currently hardcoded, but could be added to the config file in the future
  config.cacheStorageVersion = makeid(8); // changes on each build

  return config;
}

let devtool: DevTool;

if (process.env.NODE_ENV === "development") {
  devtool = "inline-source-map";
} else if (process.env.NODE_ENV === "local") {
  devtool = "source-map";
} else {
  devtool = "eval";
}

const plugins: (RspackPluginInstance | RspackPluginFunction)[] = [
  new rspack.HtmlRspackPlugin({
    template: "./index.html",
  }),
  // @ts-ignore This plugin works fine, TypeScript just doesn't realise it
  new NodePolyfillPlugin(),
  // biome-ignore lint: Name needs to be in all caps for Rspack magic to work
  new rspack.DefinePlugin({ CONFIG: JSON.stringify(readYamlConfigFile()) }),
];

const moduleRules: RuleSetRule[] = [
  {
    test: /\.html$/i,
    loader: "html-loader",
  },
  {
    test: /\.js$/,
    exclude: [/(node_modules|service-worker.js)/],
    loader: "builtin:swc-loader",
    type: "javascript/auto",
  },
  {
    test: /\.jsx$/,
    exclude: /(node_modules|service-worker.js)/,
    use: {
      loader: "builtin:swc-loader",
      options: {
        jsc: {
          parser: {
            syntax: "ecmascript",
            jsx: true,
          },
        },
      },
    },
    type: "javascript/auto",
  },
  {
    test: /\.s[ac]ss$/i,
    use: ["sass-loader"],
    type: "css/auto",
  },
  {
    test: /\.(png|svg|jpg|jpeg|gif)$/i,
    type: "asset/resource",
  },
  {
    test: /\.(woff|woff2|eot|ttf|otf)$/i,
    type: "asset/resource",
  },
  {
    test: /\.(json)$/i,
    type: "asset/resource",
  },
];

if (["production", "local"].includes(process.env.NODE_ENV ?? "")) {
  plugins.push(
    new rspack.CssExtractRspackPlugin({
      filename: "[name].[contenthash].css",
      chunkFilename: "[id].[contenthash].css",
    }),
    // @ts-ignore This plugin works fine, TypeScript just doesn't realise it
    new CompressionPlugin(),
  );

  moduleRules.push({
    test: /\.s[ac]ss$/i,
    use: ["sass-loader"],
    type: "css/auto",
  });
}

export default defineConfig({
  mode: process.env.NODE_ENV === "development" ? "development" : "production",
  entry: {
    app: "./src/index.jsx",
    "service-worker": {
      import: "./service-worker.js",
      filename: "[name].js",
    },
  },
  output: {
    filename: "[name].[contenthash].js",
    assetModuleFilename: "[name][ext]",
    path: path.resolve(__dirname, "dist"),
    publicPath: "/",
    clean: true,
  },
  devtool,
  devServer:
    process.env.NODE_ENV === "development"
      ? {
          // contentBase: './dist',
          historyApiFallback: true,
          proxy: process.env.PROXY_HOST
            ? {
                context: ["/api", "/images"],
                target: process.env.PROXY_HOST,
                secure: false, // keep false for https hosts
              }
            : undefined,
          webSocketServer: false,
        }
      : undefined,
  plugins,
  resolve: {
    extensions: [".js", ".jsx"],
  },
  module: {
    rules: moduleRules,
  },
  experiments: {
    css: true,
  },
});
