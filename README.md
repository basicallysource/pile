# pile

What a LEGO sorter sorted, as a bulk catalog: every part in every color, with counts. Then which LEGO sets those pieces make up, ranked by how complete each one is, how long a sorter would take to come across all of a set's pieces, and which bin on your shelves holds what, searchable as you type.

It reads the records of a [Basically](https://basically.website) sorter (a machine that identifies bulk LEGO piece by piece) and matches them against Rebrickable's catalog of every set LEGO sold, the separate builds within sets (a building, a vehicle), and designers' custom models.

## Run it

You need Go 1.27, [buf](https://buf.build), `protoc-gen-go`, `protoc-gen-connect-go`, and Node with corepack.

```sh
scripts/build.sh
bin/pile -data ./data -addr :8790
```

Then open <http://localhost:8790>.

pile does not run on nothing: the data folder needs Rebrickable's catalog, a sorter Hive's `parts.db`, and the records of at least one sorter. [docs/running.md](docs/running.md) says exactly what goes in it and where each piece comes from. Reading that is the whole setup.

## Where things came from

- **Catalog.** Parts, colors, sets and set inventories are [Rebrickable](https://rebrickable.com)'s CSV downloads. Part and set pictures are Rebrickable's, loaded from their site and not stored here. Free custom models (MOCs) are from Rebrickable too.
- **Builds within sets.** Read from the fan-made 3D models of sets in the [LDraw](https://www.ldraw.org) Official Model Repository, with the LDraw parts library; both are shared under CCAL 2.0, and each model's author is shown with its builds. Their pictures are drawn from those models.
- **Pieces.** The records come from the [Sorter](https://github.com/basicallysource/sorter-v2) and its Hive, which classifies in BrickLink's ids. `parts.db`, the Hive's parts catalog, maps those to Rebrickable's.
- **Design.** The look is the Sorter design system, `software/sorter-design-system` in [basicallysource/sorter-v2](https://github.com/basicallysource/sorter-v2). Its tokens and components are copied in unchanged (`app/src/app.css`, `app/src/lib/components/`). See [docs/design-system](docs/design-system/README.md).
- **Built with** Go, [Connect](https://connectrpc.com), SQLite, SvelteKit and Tailwind, and written with Claude Code.

LEGO is a trademark of the LEGO Group, which does not sponsor or endorse this project. Rebrickable and BrickLink are separate sites with their own terms.

## License

MIT, see [LICENSE](LICENSE).

## For agents

Read [AGENTS.md](AGENTS.md), then [docs/running.md](docs/running.md).
