const esbuild = require("esbuild");

async function watch() {
    let ctx = await esbuild.context({
        entryPoints: ["./frontend/admin/Admin.jsx"],
        minify: true,
        outfile: "./resources/static/admin/admin.js",
        bundle: true,
    });
    await ctx.watch();
    console.log('Watching...');
}

watch()



