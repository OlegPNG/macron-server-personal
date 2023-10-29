/** @type {import('tailwindcss').Config} */
module.exports = {
    content: ["./src/**/*.{html,js}", "./templates/**/*.html"],
    theme: {
        extend: {},
    },
    plugins: [
        require("@catppuccin/tailwindcss")({
            prefix: "cat",
            defaultFlavour: "frappe",
        }),
    ],
}
