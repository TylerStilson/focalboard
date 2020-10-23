// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
/* eslint-disable max-lines */
import React from 'react'

import {Archiver} from '../archiver'
import {BlockIcons} from '../blockIcons'
import {IPropertyOption} from '../blocks/board'
import {Card, MutableCard} from '../blocks/card'
import {BoardTree} from '../viewModel/boardTree'
import {CardFilter} from '../cardFilter'
import ViewMenu from '../components/viewMenu'
import {Constants} from '../constants'
import {Menu as OldMenu} from '../menu'
import mutator from '../mutator'
import {OctoUtils} from '../octoUtils'
import {Utils} from '../utils'
import Menu from '../widgets/menu'
import MenuWrapper from '../widgets/menuWrapper'

import {BoardCard} from './boardCard'
import {BoardColumn} from './boardColumn'
import Button from './button'
import {CardDialog} from './cardDialog'
import {Editable} from './editable'
import RootPortal from './rootPortal'

type Props = {
    boardTree?: BoardTree
    showView: (id: string) => void
    showFilter: (el: HTMLElement) => void
    setSearchText: (text: string) => void
}

type State = {
    isSearching: boolean
    shownCard?: Card
    viewMenu: boolean
    isHoverOnCover: boolean
    selectedCards: Card[]
}

class BoardComponent extends React.Component<Props, State> {
    private draggedCards: Card[] = []
    private draggedHeaderOption: IPropertyOption
    private backgroundRef = React.createRef<HTMLDivElement>()
    private searchFieldRef = React.createRef<Editable>()

    private keydownHandler = (e: KeyboardEvent) => {
        if (e.target !== document.body) {
            return
        }

        if (e.keyCode === 27) {
            if (this.state.selectedCards.length > 0) {
                this.setState({selectedCards: []})
                e.stopPropagation()
            }
        }
    }

    componentDidMount(): void {
        document.addEventListener('keydown', this.keydownHandler)
    }

    componentWillUnmount(): void {
        document.removeEventListener('keydown', this.keydownHandler)
    }

    constructor(props: Props) {
        super(props)
        this.state = {
            isHoverOnCover: false,
            isSearching: Boolean(this.props.boardTree?.getSearchText()),
            viewMenu: false,
            selectedCards: [],
        }
    }

    shouldComponentUpdate(): boolean {
        return true
    }

    componentDidUpdate(prevPros: Props, prevState: State): void {
        if (this.state.isSearching && !prevState.isSearching) {
            this.searchFieldRef.current.focus()
        }
    }

    render(): JSX.Element {
        const {boardTree, showView} = this.props

        if (!boardTree || !boardTree.board) {
            return (
                <div>Loading...</div>
            )
        }

        const propertyValues = boardTree.groupByProperty?.options || []
        Utils.log(`${propertyValues.length} propertyValues`)

        const groupByStyle = {color: '#000000'}
        const {board, activeView} = boardTree
        const visiblePropertyTemplates = board.cardProperties.filter((template) => activeView.visiblePropertyIds.includes(template.id))
        const hasFilter = activeView.filter && activeView.filter.filters?.length > 0
        const hasSort = activeView.sortOptions.length > 0
        const visibleGroups = boardTree.groups.filter(group => !group.isHidden)
        const hiddenGroups = boardTree.groups.filter(group => group.isHidden)

        return (
            <div
                className='octo-app'
                ref={this.backgroundRef}
                onClick={(e) => {
                    this.backgroundClicked(e)
                }}
            >
                {this.state.shownCard &&
                <RootPortal>
                    <CardDialog
                        boardTree={boardTree}
                        card={this.state.shownCard}
                        onClose={() => this.setState({shownCard: undefined})}
                    />
                </RootPortal>}

                <div className='octo-frame'>
                    <div
                        className='octo-hovercontrols'
                        onMouseOver={() => {
                            this.setState({...this.state, isHoverOnCover: true})
                        }}
                        onMouseLeave={() => {
                            this.setState({...this.state, isHoverOnCover: false})
                        }}
                    >
                        <Button
                            style={{display: (!board.icon && this.state.isHoverOnCover) ? null : 'none'}}
                            onClick={() => {
                                const newIcon = BlockIcons.shared.randomIcon()
                                mutator.changeIcon(board, newIcon)
                            }}
                        >Add Icon</Button>
                    </div>

                    <div className='octo-icontitle'>
                        {board.icon ?
                            <MenuWrapper>
                                <div className='octo-button octo-icon'>{board.icon}</div>
                                <Menu>
                                    <Menu.Text
                                        id='random'
                                        name='Random'
                                        onClick={() => mutator.changeIcon(board, BlockIcons.shared.randomIcon())}
                                    />
                                    <Menu.Text
                                        id='remove'
                                        name='Remove Icon'
                                        onClick={() => mutator.changeIcon(board, undefined, 'remove icon')}
                                    />
                                </Menu>
                            </MenuWrapper> :
                            undefined}
                        <Editable
                            className='title'
                            text={board.title}
                            placeholderText='Untitled Board'
                            onChanged={(text) => {
                                mutator.changeTitle(board, text)
                            }}
                        />
                    </div>

                    <div className='octo-board'>
                        <div className='octo-controls'>
                            <Editable
                                style={{color: '#000000', fontWeight: 600}}
                                text={activeView.title}
                                placeholderText='Untitled View'
                                onChanged={(text) => {
                                    mutator.changeTitle(activeView, text)
                                }}
                            />
                            <MenuWrapper>
                                <div
                                    className='octo-button'
                                    style={{color: '#000000', fontWeight: 600}}
                                >
                                    <div className='imageDropdown'/>
                                </div>
                                <ViewMenu
                                    board={board}
                                    boardTree={boardTree}
                                    showView={showView}
                                />
                            </MenuWrapper>
                            <div className='octo-spacer'/>
                            <div
                                className='octo-button'
                                onClick={(e) => {
                                    this.propertiesClicked(e)
                                }}
                            >Properties</div>
                            <div
                                className='octo-button'
                                id='groupByButton'
                                onClick={(e) => {
                                    this.groupByClicked(e)
                                }}
                            >
                                Group by <span
                                    style={groupByStyle}
                                    id='groupByLabel'
                                         >{boardTree.groupByProperty?.name}</span>
                            </div>
                            <div
                                className={hasFilter ? 'octo-button active' : 'octo-button'}
                                onClick={(e) => {
                                    this.filterClicked(e)
                                }}
                            >Filter</div>
                            <div
                                className={hasSort ? 'octo-button active' : 'octo-button'}
                                onClick={(e) => {
                                    OctoUtils.showSortMenu(e, boardTree)
                                }}
                            >Sort</div>
                            {this.state.isSearching ?
                                <Editable
                                    ref={this.searchFieldRef}
                                    text={boardTree.getSearchText()}
                                    placeholderText='Search text'
                                    style={{color: '#000000'}}
                                    onChanged={(text) => {
                                        this.searchChanged(text)
                                    }}
                                    onKeyDown={(e) => {
                                        this.onSearchKeyDown(e)
                                    }}
                                /> :
                                <div
                                    className='octo-button'
                                    onClick={() => {
                                        this.setState({...this.state, isSearching: true})
                                    }}
                                >Search</div>
                            }
                            <div
                                className='octo-button'
                                onClick={(e) => {
                                    this.optionsClicked(e)
                                }}
                            ><div className='imageOptions'/></div>
                            <div
                                className='octo-button filled'
                                onClick={() => {
                                    this.addCard(undefined)
                                }}
                            >New</div>
                        </div>

                        {/* Headers */}

                        <div
                            className='octo-board-header'
                            id='mainBoardHeader'
                        >

                            {/* No value */}

                            <div className='octo-board-header-cell'>
                                <div
                                    className='octo-label'
                                    title={`Items with an empty ${boardTree.groupByProperty?.name} property will go here. This column cannot be removed.`}
                                >{`No ${boardTree.groupByProperty?.name}`}</div>
                                <Button>{`${boardTree.emptyGroupCards.length}`}</Button>
                                <div className='octo-spacer'/>
                                <Button><div className='imageOptions'/></Button>
                                <Button
                                    onClick={() => {
                                        this.addCard(undefined)
                                    }}
                                ><div className='imageAdd'/></Button>
                            </div>

                            {/* Visible column headers */}

                            {visibleGroups.map((group) =>
                                (<div
                                    key={group.option.id}
                                    className='octo-board-header-cell'

                                    draggable={true}
                                    onDragStart={() => {
                                        this.draggedHeaderOption = group.option
                                    }}
                                    onDragEnd={() => {
                                        this.draggedHeaderOption = undefined
                                    }}

                                    onDragOver={(e) => {
                                        e.preventDefault(); (e.target as HTMLElement).classList.add('dragover')
                                    }}
                                    onDragEnter={(e) => {
                                        e.preventDefault(); (e.target as HTMLElement).classList.add('dragover')
                                    }}
                                    onDragLeave={(e) => {
                                        e.preventDefault(); (e.target as HTMLElement).classList.remove('dragover')
                                    }}
                                    onDrop={(e) => {
                                        e.preventDefault(); (e.target as HTMLElement).classList.remove('dragover'); this.onDropToColumn(group.option)
                                    }}
                                >
                                    <Editable
                                        className={`octo-label ${group.option.color}`}
                                        text={group.option.value}
                                        onChanged={(text) => {
                                            this.propertyNameChanged(group.option, text)
                                        }}
                                    />
                                    <Button>{`${group.cards.length}`}</Button>
                                    <div className='octo-spacer'/>
                                    <MenuWrapper>
                                        <Button><div className='imageOptions'/></Button>
                                        <Menu>
                                            <Menu.Text
                                                id='hide'
                                                name='Hide'
                                                onClick={() => mutator.hideViewColumn(activeView, group.option.id)}
                                            />
                                            <Menu.Text
                                                id='delete'
                                                name='Delete'
                                                onClick={() => mutator.deletePropertyOption(boardTree, boardTree.groupByProperty, group.option)}
                                            />
                                            <Menu.Separator/>
                                            {Constants.menuColors.map((color) =>
                                                (<Menu.Color
                                                    key={color.id}
                                                    id={color.id}
                                                    name={color.name}
                                                    onClick={() => mutator.changePropertyOptionColor(boardTree.board, boardTree.groupByProperty, group.option, color.id)}
                                                />),
                                            )}
                                        </Menu>
                                    </MenuWrapper>
                                    <Button
                                        onClick={() => {
                                            this.addCard(group.option.id)
                                        }}
                                    ><div className='imageAdd'/></Button>
                                </div>),
                            )}

                            {/* Hidden column header */}

                            {(() => {
                                if (hiddenGroups.length > 0) {
                                    return <div className='octo-board-header-cell narrow'>Hidden columns</div>
                                }
                            })()}

                            <div className='octo-board-header-cell narrow'>
                                <Button
                                    onClick={(e) => {
                                        this.addGroupClicked()
                                    }}
                                >+ Add a group</Button>
                            </div>
                        </div>

                        {/* Main content */}

                        <div
                            className='octo-board-body'
                            id='mainBoardBody'
                        >

                            {/* No value column */}

                            <BoardColumn
                                onDrop={(e) => {
                                    this.onDropToColumn(undefined)
                                }}
                            >
                                {boardTree.emptyGroupCards.map((card) =>
                                    (<BoardCard
                                        card={card}
                                        visiblePropertyTemplates={visiblePropertyTemplates}
                                        key={card.id}
                                        isSelected={this.state.selectedCards.includes(card)}
                                        onClick={(e) => {
                                            this.cardClicked(e, card)
                                        }}
                                        onDragStart={() => {
                                            this.draggedCards = this.state.selectedCards.includes(card) ? this.state.selectedCards : [card]
                                        }}
                                        onDragEnd={() => {
                                            this.draggedCards = []
                                        }}
                                    />),
                                )}
                                <Button
                                    onClick={() => {
                                        this.addCard(undefined)
                                    }}
                                >+ New</Button>
                            </BoardColumn>

                            {/* Columns */}

                            {visibleGroups.map((group) =>
                                (<BoardColumn
                                    onDrop={(e) => {
                                        this.onDropToColumn(group.option)
                                    }}
                                    key={group.option.id}
                                >
                                    {group.cards.map((card) =>
                                        (<BoardCard
                                            card={card}
                                            visiblePropertyTemplates={visiblePropertyTemplates}
                                            key={card.id}
                                            isSelected={this.state.selectedCards.includes(card)}
                                            onClick={(e) => {
                                                this.cardClicked(e, card)
                                            }}
                                            onDragStart={() => {
                                                this.draggedCards = this.state.selectedCards.includes(card) ? this.state.selectedCards : [card]
                                            }}
                                            onDragEnd={() => {
                                                this.draggedCards = []
                                            }}
                                        />),
                                    )}
                                    <Button
                                        onClick={() => {
                                            this.addCard(group.option.id)
                                        }}
                                    >+ New</Button>
                                </BoardColumn>),
                            )}

                            {/* Hidden columns */}

                            {(() => {
                                if (hiddenGroups.length > 0) {
                                    return(
                                        <div className='octo-board-column narrow'>
                                            {hiddenGroups.map((group) =>
                                                <MenuWrapper key={group.option.id}>
                                                    <div
                                                        key={group.option.id}
                                                        className={`octo-label ${group.option.color}`}
                                                    >
                                                        {group.option.value}
                                                    </div>
                                                <Menu>
                                                    <Menu.Text
                                                        id='show'
                                                        name='Show'
                                                        onClick={() => mutator.unhideViewColumn(activeView, group.option.id)}
                                                    />
                                                </Menu>
                                            </MenuWrapper>
                                            )}
                                        </div>
                                    )
                                }
                            })()}

                        </div>
                    </div>
                </div>
            </div>
        )
    }

    private backgroundClicked(e: React.MouseEvent) {
        if (this.state.selectedCards.length > 0) {
            this.setState({selectedCards: []})
            e.stopPropagation()
        }
    }

    private async addCard(groupByOptionId?: string): Promise<void> {
        const {boardTree} = this.props
        const {activeView, board} = boardTree

        const card = new MutableCard()
        card.parentId = boardTree.board.id
        card.properties = CardFilter.propertiesThatMeetFilterGroup(activeView.filter, board.cardProperties)
        card.icon = BlockIcons.shared.randomIcon()
        if (boardTree.groupByProperty) {
            card.properties[boardTree.groupByProperty.id] = groupByOptionId
        }
        await mutator.insertBlock(card, 'add card', async () => {
            this.setState({shownCard: card})
        }, async () => {
            this.setState({shownCard: undefined})
        })
    }

    private async propertyNameChanged(option: IPropertyOption, text: string): Promise<void> {
        const {boardTree} = this.props

        await mutator.changePropertyOptionValue(boardTree, boardTree.groupByProperty, option, text)
    }

    private filterClicked(e: React.MouseEvent) {
        this.props.showFilter(e.target as HTMLElement)
    }

    private async optionsClicked(e: React.MouseEvent) {
        const {boardTree} = this.props

        OldMenu.shared.options = [
            {id: 'exportBoardArchive', name: 'Export board archive'},
            {id: 'testAdd100Cards', name: 'TEST: Add 100 cards'},
            {id: 'testAdd1000Cards', name: 'TEST: Add 1,000 cards'},
            {id: 'testRandomizeIcons', name: 'TEST: Randomize icons'},
        ]

        OldMenu.shared.onMenuClicked = async (id: string) => {
            switch (id) {
            case 'exportBoardArchive': {
                Archiver.exportBoardTree(boardTree)
                break
            }
            case 'testAdd100Cards': {
                this.testAddCards(100)
                break
            }
            case 'testAdd1000Cards': {
                this.testAddCards(1000)
                break
            }
            case 'testRandomizeIcons': {
                this.testRandomizeIcons()
                break
            }
            }
        }
        OldMenu.shared.showAtElement(e.target as HTMLElement)
    }

    private async testAddCards(count: number) {
        const {boardTree} = this.props
        const {board, activeView} = boardTree

        const startCount = boardTree?.cards?.length
        let optionIndex = 0

        for (let i = 0; i < count; i++) {
            const card = new MutableCard()
            card.parentId = boardTree.board.id
            card.properties = CardFilter.propertiesThatMeetFilterGroup(activeView.filter, board.cardProperties)
            if (boardTree.groupByProperty && boardTree.groupByProperty.options.length > 0) {
                // Cycle through options
                const option = boardTree.groupByProperty.options[optionIndex]
                optionIndex = (optionIndex + 1) % boardTree.groupByProperty.options.length
                card.properties[boardTree.groupByProperty.id] = option.id
                card.title = `Test Card ${startCount + i + 1}`
                card.icon = BlockIcons.shared.randomIcon()
            }
            await mutator.insertBlock(card, 'test add card')
        }
    }

    private async testRandomizeIcons() {
        const {boardTree} = this.props

        for (const card of boardTree.cards) {
            mutator.changeIcon(card, BlockIcons.shared.randomIcon(), 'randomize icon')
        }
    }

    private async propertiesClicked(e: React.MouseEvent) {
        const {boardTree} = this.props
        const {activeView} = boardTree

        const selectProperties = boardTree.board.cardProperties
        OldMenu.shared.options = selectProperties.map((o) => {
            const isVisible = activeView.visiblePropertyIds.includes(o.id)
            return {id: o.id, name: o.name, type: 'switch', isOn: isVisible}
        })

        OldMenu.shared.onMenuToggled = async (id: string, isOn: boolean) => {
            const property = selectProperties.find((o) => o.id === id)
            Utils.assertValue(property)
            Utils.log(`Toggle property ${property.name} ${isOn}`)

            let newVisiblePropertyIds = []
            if (activeView.visiblePropertyIds.includes(id)) {
                newVisiblePropertyIds = activeView.visiblePropertyIds.filter((o) => o !== id)
            } else {
                newVisiblePropertyIds = [...activeView.visiblePropertyIds, id]
            }
            await mutator.changeViewVisibleProperties(activeView, newVisiblePropertyIds)
        }
        OldMenu.shared.showAtElement(e.target as HTMLElement)
    }

    private async groupByClicked(e: React.MouseEvent) {
        const {boardTree} = this.props

        const selectProperties = boardTree.board.cardProperties.filter((o) => o.type === 'select')
        OldMenu.shared.options = selectProperties.map((o) => {
            return {id: o.id, name: o.name}
        })
        OldMenu.shared.onMenuClicked = async (command: string) => {
            if (boardTree.activeView.groupById === command) {
                return
            }

            await mutator.changeViewGroupById(boardTree.activeView, command)
        }
        OldMenu.shared.showAtElement(e.target as HTMLElement)
    }

    private cardClicked(e: React.MouseEvent, card: Card): void {
        if (e.shiftKey) {
            // Shift+Click = add to selection
            let selectedCards = this.state.selectedCards.slice()
            if (selectedCards.includes(card)) {
                selectedCards = selectedCards.filter((o) => o != card)
            } else {
                selectedCards.push(card)
            }
            this.setState({selectedCards})
        } else {
            this.setState({selectedCards: [], shownCard: card})
        }

        e.stopPropagation()
    }

    private async addGroupClicked() {
        Utils.log('onAddGroupClicked')

        const {boardTree} = this.props

        const option: IPropertyOption = {
            id: Utils.createGuid(),
            value: 'New group',
            color: 'propColorDefault',
        }

        Utils.assert(boardTree.groupByProperty)
        await mutator.insertPropertyOption(boardTree, boardTree.groupByProperty, option, 'add group')
    }

    private async onDropToColumn(option: IPropertyOption) {
        const {boardTree} = this.props
        const {draggedCards, draggedHeaderOption} = this
        const optionId = option ? option.id : undefined

        Utils.assertValue(mutator)
        Utils.assertValue(boardTree)

        if (draggedCards.length > 0) {
            for (const draggedCard of draggedCards) {
                Utils.log(`ondrop. Card: ${draggedCard.title}, column: ${optionId}`)
                const oldValue = draggedCard.properties[boardTree.groupByProperty.id]
                if (optionId !== oldValue) {
                    await mutator.changePropertyValue(draggedCard, boardTree.groupByProperty.id, optionId, 'drag card')
                }
            }
        } else if (draggedHeaderOption) {
            Utils.log(`ondrop. Header option: ${draggedHeaderOption.value}, column: ${option?.value}`)
            Utils.assertValue(boardTree.groupByProperty)

            // Move option to new index
            const {board} = boardTree
            const options = boardTree.groupByProperty.options
            const destIndex = option ? options.indexOf(option) : 0

            await mutator.changePropertyOptionOrder(board, boardTree.groupByProperty, draggedHeaderOption, destIndex)
        }
    }

    private onSearchKeyDown(e: React.KeyboardEvent) {
        if (e.keyCode === 27) { // ESC: Clear search
            this.searchFieldRef.current.text = ''
            this.setState({isSearching: false})
            this.props.setSearchText(undefined)
            e.preventDefault()
        }
    }

    private searchChanged(text?: string) {
        this.props.setSearchText(text)
    }
}

export {BoardComponent}
