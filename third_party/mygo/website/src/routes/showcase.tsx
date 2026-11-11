import { createFileRoute } from "@tanstack/react-router"
import { ArrowUpRightIcon } from "lucide-react"

import { site } from "@/lib/site"
import { showcase, type ShowcaseApp } from "@/showcase"

const title = "Showcase · MyGo"
const description = "Apps built with MyGo by the community, with native UI or a web frontend."
const dataFile = `${site.repo}/blob/main/website/src/showcase.ts`

const modes = { native: "Native UI", web: "Web frontend" } as const

export const Route = createFileRoute("/showcase")({
  head: () => ({
    meta: [
      { title },
      { name: "description", content: description },
      { property: "og:title", content: title },
      { property: "og:description", content: description },
    ],
  }),
  component: Showcase,
})

function Showcase() {
  return (
    <main className="flex-1">
      <section className="border-b px-4 pt-16 pb-12 sm:px-10 md:pt-20">
        <p className="label">Showcase</p>
        <h1 className="mt-5 text-3xl leading-[1.1] font-semibold tracking-[-0.035em] sm:text-4xl">Built with MyGo.</h1>
        <p className="mt-5 max-w-2xl text-base leading-7 text-muted-foreground">
          Apps the community has made with MyGo, drawing native UI or showing a web frontend.
        </p>
        <p className="mt-6 text-sm">
          <a href={dataFile} className="font-medium underline decoration-gopher/45 underline-offset-[5px] hover:decoration-gopher">
            Add your app
          </a>
        </p>
      </section>

      <div className="-mr-px grid sm:grid-cols-2 lg:grid-cols-3">
        {showcase.map((app) => (
          <Card key={app.url} app={app} />
        ))}
      </div>

      <section className="px-4 py-10 sm:px-10">
        <h2 className="text-xl font-semibold tracking-tight">Add your app</h2>
        <p className="mt-3 max-w-2xl text-sm leading-6 text-muted-foreground">
          Built something with MyGo? Open a pull request that adds it to{" "}
          <a href={dataFile} className="font-mono text-[13px] text-foreground underline underline-offset-4">
            website/src/showcase.ts
          </a>
          , with its name, a short description, a link, and whether it draws native UI or shows a web frontend. A screenshot is optional.
        </p>
      </section>
    </main>
  )
}

function Card({ app }: { app: ShowcaseApp }) {
  return (
    <article className="group relative flex min-w-0 flex-col border-r border-b">
      {app.image && <img src={app.image} alt="" loading="lazy" className="aspect-[16/10] w-full border-b object-cover object-top" />}
      <div className="flex flex-1 flex-col gap-3 px-4 py-5 sm:px-6">
        <header className="flex items-start justify-between gap-4">
          <h2 className="min-w-0 font-semibold tracking-tight">
            <a href={app.url} className="after:absolute after:inset-0 group-hover:underline group-hover:underline-offset-4">
              {app.name}
            </a>
          </h2>
          <ArrowUpRightIcon className="size-4 shrink-0 text-muted-foreground transition-colors group-hover:text-foreground" />
        </header>
        <p className="text-sm leading-6 text-muted-foreground">{app.description}</p>
        {(app.mode || app.author) && (
          <p className="label mt-auto flex flex-wrap gap-x-4 gap-y-1 pt-1">
            {app.mode && <span>{modes[app.mode]}</span>}
            {app.author && <span>By {app.author}</span>}
          </p>
        )}
      </div>
    </article>
  )
}
