import { Link, useLocation } from "@tanstack/react-router"
import { MenuIcon } from "lucide-react"

import { GitHubIcon } from "@/components/icons"
import { Wordmark } from "@/components/logo"
import { Search } from "@/components/search"
import { ThemeToggle } from "@/components/theme-toggle"
import { Button, buttonVariants } from "@/components/ui/button"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { bookOf, slugOfPath } from "@/lib/docs"
import { site } from "@/lib/site"

const navLink = "text-muted-foreground transition-colors hover:text-foreground data-current:text-foreground"
const mobileNavLink = "min-h-11 px-3 data-current:bg-accent aria-[current=page]:bg-accent"

export function SiteHeader() {
  // The docs and native UI's are apart, each with its own sidebar.
  const book = useLocation({ select: (l) => (l.pathname.startsWith("/docs") ? bookOf(slugOfPath(l.pathname)) : undefined) })
  return (
    <header className="frame sticky top-0 z-40 flex h-14 items-center gap-6 border-b bg-background/80 px-4 backdrop-blur-xl sm:px-6">
      <Link to="/">
        <Wordmark />
      </Link>
      <nav className="hidden items-center gap-5 text-sm sm:flex">
        <Link to="/docs" className={navLink} activeOptions={{ exact: true }} data-current={book === "docs" || undefined}>
          Docs
        </Link>
        <Link to="/docs/$" params={{ _splat: "ui" }} className={navLink} activeOptions={{ exact: true }} data-current={book === "ui" || undefined}>
          UI
        </Link>
        <a href={`${site.repo}/tree/main/examples`} className={navLink}>
          Examples
        </a>
        <Link to="/benchmarks" className={navLink}>
          Benchmarks
        </Link>
      </nav>
      <div className="ml-auto flex items-center gap-1">
        <Search />
        <a href={site.repo} aria-label="GitHub" className={buttonVariants({ variant: "ghost", size: "icon" })}>
          <GitHubIcon />
        </a>
        <ThemeToggle />
        <DropdownMenu modal={false}>
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon" className="sm:hidden" aria-label="Navigation menu" />}>
            <MenuIcon />
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-44 sm:hidden">
            <DropdownMenuItem
              className={mobileNavLink}
              render={<Link to="/docs" activeOptions={{ exact: true }} data-current={book === "docs" || undefined} />}
            >
              Docs
            </DropdownMenuItem>
            <DropdownMenuItem
              className={mobileNavLink}
              render={<Link to="/docs/$" params={{ _splat: "ui" }} activeOptions={{ exact: true }} data-current={book === "ui" || undefined} />}
            >
              UI
            </DropdownMenuItem>
            <DropdownMenuItem className={mobileNavLink} render={<a href={`${site.repo}/tree/main/examples`} />}>
              Examples
            </DropdownMenuItem>
            <DropdownMenuItem className={mobileNavLink} render={<Link to="/benchmarks" />}>
              Benchmarks
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  )
}
